package module

import (
	"errors"
	"fmt"
	"github.com/pdbogen/gosible/transport"
	"github.com/pdbogen/gosible/types"
	"strings"
)

type Cmd struct {
	cmd string
}

func (*Cmd) Always() bool { return false }

func (c *Cmd) Configure(_ *types.Target, params map[string]string) error {
	if cmd, ok := params["cmd"]; !ok {
		return errors.New("cmd called with no `cmd` set")
	} else {
		c.cmd = cmd
	}
	return nil
}

func (c *Cmd) Execute(target *types.Target, transport transport.Transport) (change bool, err error) {
	stdout, stderr, res, err := transport.Do([]string{
		"sh", "-c", c.cmd,
	})
	if err != nil {
		return false, fmt.Errorf("error running command: %s", err)
	}

	for _, line := range strings.Split(string(stdout), "\n") {
		log.Debugf("cmd:out: %s", line)
	}

	for _, line := range strings.Split(string(stderr), "\n") {
		log.Debugf("cmd:err: %s", line)
	}

	if res != 0 {
		return false, fmt.Errorf("non-zero running command %s: %d", c.cmd, res)
	}

	return true, nil
}

func (c *Cmd) Name() string {
	return "cmd"
}

var _ Module = (*Cmd)(nil)
