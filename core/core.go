package core

import (
	"errors"
	"fmt"
	"github.com/op/go-logging"
	"github.com/pdbogen/gosible/module"
	"github.com/pdbogen/gosible/transport"
	"github.com/pdbogen/gosible/types"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

var log = logging.MustGetLogger("gosible/core")

type Core struct {
	Root           string
	CredentialFile string
	TargetFile     string
	Credentials    []*types.Credential
	Targets        []*types.Target
	sets           []*types.Set
	setMap         map[string]*types.Set
	Modules        map[string]module.Module
	Transports     map[string]transport.TransportConnect
	log            *logging.Logger
	Registry       map[string]bool
	Execs          int64
	Changes        int64
}

func (c *Core) Load() error {
	if err := c.loadCredentials(); err != nil {
		return fmt.Errorf("loading credentials: %s", err)
	}
	if err := c.loadTasks(); err != nil {
		return fmt.Errorf("loading tasks: %s", err)
	}
	if err := c.loadTargets(); err != nil {
		return fmt.Errorf("loading targets: %s", err)
	}
	if err := c.hydrateSets(); err != nil {
		return fmt.Errorf("hydrating tasks: %s", err)
	}
	return nil
}

func (c *Core) checkTask(task *types.Task) error {
	if len(task.Modules) == 0 {
		return errors.New("no modules")
	}
	if len(task.Modules) > 1 {
		return errors.New("multiple modules")
	}
	return nil
}

func (c *Core) hydrateSets() error {
	return c.hydrateSetsCount(0)
}

func (c *Core) hydrateSetsCount(preCount int) error {
	if c.setMap == nil {
		c.setMap = map[string]*types.Set{}
	}
	for _, ts := range c.sets {
		if _, ok := c.setMap[ts.Name]; ok {
			return fmt.Errorf("%s has duplicate name", ts.Name)
		}
		for taskIdx, task := range ts.Tasks {
			name := "task"
			if task.Name != "" {
				name = task.Name
			}
			if err := c.checkTask(task); err != nil {
				return fmt.Errorf("%s/%s (%d) is invalid: %s", ts.Name, name, taskIdx, err)
			}
		}
		c.setMap[ts.Name] = ts
	}
	return nil
}

func (c *Core) loadTasks() error {
	path := c.pathForFile(c.TargetFile, "sets.yml")
	taskYaml, err := c.readFile(c.TargetFile, "sets.yml")
	if err != nil {
		return fmt.Errorf("reading %s: %s", path, err)
	}

	c.sets = []*types.Set{}
	if err := yaml.Unmarshal(taskYaml, &c.sets); err != nil {
		return fmt.Errorf("parsing %s: %s", path, err)
	}
	log.Debugf("unmarshalled %d tasks from %s", len(c.sets), path)
	return nil
}

// Returns the credential named by Target; or nil, if no credential was requested. If an error occurs, error will be
// non-nil and credential will be nil.
func (c *Core) credentialForTarget(t *types.Target) (*types.Credential, error) {
	if t == nil {
		return nil, fmt.Errorf("credentialForTarget called with nil target")
	}
	if t.CredentialName == "" {
		log.Warningf("target %s did not request a credential", t.Name)
		return nil, nil
	}

	for i, cred := range c.Credentials {
		if cred.Name == t.CredentialName {
			return c.Credentials[i], nil
		}
	}
	return nil, fmt.Errorf("no credential '%s' for target %s", t.CredentialName, t.Name)
}

func (c *Core) populateModules() {
	if c.Modules == nil {
		c.Modules = map[string]module.Module{
			"cmd":     &module.Cmd{},
			"file":    &module.File{},
			"package": &module.Package{},
			"survey":  &module.Survey{},
		}
	}
}

func (c *Core) populateTransports() {
	if c.Transports == nil {
		c.Transports = map[string]transport.TransportConnect{
			"SSH": (*transport.SSH)(nil).Connect,
		}
	}
}

func (c *Core) transportForTarget(t *types.Target) (transport.Transport, error) {
	cred, err := c.credentialForTarget(t)
	if err != nil {
		return nil, fmt.Errorf("obtaining transport: %s", err)
	}

	c.populateTransports()

	if t.TransportName == "" {
		t.TransportName = "SSH"
	}

	for trName, trConn := range c.Transports {
		if t.TransportName == trName {
			return trConn(t, cred)
		}
	}
	return nil, fmt.Errorf("no transport '%s' for target %s", t.TransportName, t.Name)
}

func (c *Core) register(params map[string]string) {
	if c.Registry == nil {
		c.Registry = map[string]bool{}
	}
	if register, ok := params["register"]; ok {
		c.Registry[register] = true
	}
}

func (c *Core) checkWhen(params map[string]string) bool {
	if when, ok := params["when"]; ok {
		if c.Registry == nil {
			return false
		}
		result := false
		ors := strings.Split(when, " or ")
		for _, or := range ors {
			conj := true
			ands := strings.Split(or, " and ")
			for _, cond := range ands {
				if len(cond) > 1 && cond[0] == '!' {
					if res, ok := c.Registry[cond[1:]]; ok && res {
						conj = false
						break
					}
				} else {
					if res, ok := c.Registry[cond]; !ok || !res {
						conj = false
						break
					}
				}
			}
			result = result || conj
		}
		return result
	}
	return true
}

func (c *Core) runSet(target *types.Target, transport transport.Transport, set *types.Set) (bool, error) {
	setChange := false
tasks:
	for taskIdx, task := range set.Tasks {
		name := "task"
		if task.Name != "" {
			name = task.Name
		}
		if err := c.checkTask(task); err != nil {
			log.Warningf("%s/%s/%s (%d) skipped with: %s", target.Name, set.Name, name, taskIdx)
			continue
		}
		if params, ok := task.Modules["set"]; ok {
			recurName, ok := params["name"]
			if !ok {
				log.Warningf("%s/%s/%s (%d) does not define target task set name", target.Name, set.Name, name, taskIdx)
				continue
			}
			recurSet, ok := c.setMap[recurName]
			if !ok {
				log.Warningf("%s/%s/%s (%d) defines nonexistent task set name %s", target.Name, set.Name, name, taskIdx, recurName)
				continue
			}
			if c.checkWhen(params) {
				log.Debugf("%s/%s/%s (%d) running set '%s' by-reference", target.Name, set.Name, name, taskIdx, recurName)
				c.Execs++
				change, err := c.runSet(target, transport, recurSet)
				if err != nil {
					log.Warningf("%s/%s/%s (%d) task set %s failed: %s", target.Name, set.Name, name, taskIdx, recurName, err)
					continue
				}
				if change {
					setChange = true
					c.Changes++
					c.register(params)
				}
			}
		} else {
			for moduleName, params := range task.Modules {
				moduleObj, ok := c.Modules[moduleName]
				if !ok {
					log.Warningf("%s/%s (%d)/%s: skipping nonexistent module", target.Name, name, taskIdx, moduleName)
					continue tasks
				}
				if !c.checkWhen(params) {
					log.Debugf("%s/%s (%d)/%s: skipping", target.Name, name, taskIdx, moduleName)
					continue tasks
				}
				log.Debugf("%s/%s (%d)/%s: running", target.Name, name, taskIdx, moduleName)
				c.Execs++
				err := moduleObj.Configure(target, params)
				if err != nil {
					log.Warningf("%s/%s (%d)/%s could not configure: %s", target.Name, name, taskIdx, moduleName, err)
					continue tasks
				}
				change, err := moduleObj.Execute(target, transport)
				if change {
					c.Changes++
					c.register(params)
				}
			}
		}
	}
	return setChange, nil
}

func (c *Core) runTarget(target *types.Target) {
	if target.Metadata == nil {
		target.Metadata = map[string]string{}
	}
	target.Metadata["rootpath"] = c.Root
	tr, err := c.transportForTarget(target)
	if err != nil {
		log.Warningf("failed getting transport to target %s: %s", target.Name, err)
		return
	}
	defer tr.Close()
	c.runSet(target, tr, &types.Set{"implicit", target.Tasks})
}

func (c *Core) Run() error {
	c.populateTransports()
	c.populateModules()

	if c.Modules == nil || len(c.Modules) == 0 {
		log.Warningf("no module defined in Core")
		return nil
	}

	for _, target := range c.Targets {
		log.Debugf("Beginning target %s", target.Name)
		c.runTarget(target)
		log.Debugf("Done with target %s", target.Name)
	}

	return nil
}

func (c *Core) pathForFile(path, defaultPath string) string {
	if c.Root != "" {
		c.Root = strings.TrimRight(c.Root, "/") + "/"
	}
	if path == "" {
		return c.Root + defaultPath
	}
	return c.Root + path
}

func (c *Core) openFile(path, defaultPath string) (io.ReadCloser, error) {
	actual_path := c.pathForFile(path, defaultPath)
	f, err := os.Open(actual_path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %s", actual_path, err)
	}
	log.Debugf("opened %s", actual_path)
	return f, nil
}

func (c *Core) readFile(path, defaultPath string) ([]byte, error) {
	f, err := c.openFile(path, defaultPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	actual_path := c.pathForFile(path, defaultPath)
	content, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %s", actual_path, err)
	}
	log.Debugf("read %d bytes from %s", len(content), actual_path)
	return content, nil
}

func (c *Core) loadCredentials() error {

	path := c.pathForFile(c.CredentialFile, "credentials.yml")
	credentialYaml, err := c.readFile(c.CredentialFile, "credentials.yml")
	if err != nil {
		return fmt.Errorf("reading %s: %s", path, err)
	}

	c.Credentials = []*types.Credential{}
	if err := yaml.Unmarshal(credentialYaml, &c.Credentials); err != nil {
		return fmt.Errorf("parsing %s: %s", path, err)
	}
	log.Debugf("unmarshalled %d credentials from %s", len(c.Credentials), path)

	return nil
}

// loads the targets from the configured location
func (c *Core) loadTargets() error {
	path := c.pathForFile(c.TargetFile, "targets.yml")
	targetYaml, err := c.readFile(c.TargetFile, "targets.yml")
	if err != nil {
		return fmt.Errorf("reading %s: %s", path, err)
	}

	c.Targets = []*types.Target{}
	if err := yaml.Unmarshal(targetYaml, &c.Targets); err != nil {
		return fmt.Errorf("parsing %s: %s", path, err)
	}
	log.Debugf("unmarshalled %d targets from %s", len(c.Targets), path)
	return nil
}
