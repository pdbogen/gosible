package module

import (
	"bytes"
	"fmt"
	"github.com/pdbogen/gosible/transport"
	"github.com/pdbogen/gosible/types"
	"strings"
)

type Package struct {
	adds    []string
	removes []string
}

func (p *Package) Configure(_ *types.Target, params map[string]string) error {
	*p = Package{}

	if install, ok := params["install"]; ok {
		p.adds = strings.Split(install, " ")
	}

	if remove, ok := params["remove"]; ok {
		p.removes = strings.Split(remove, " ")
	}

	if p.adds != nil && p.removes != nil {
		pkgs := map[string]bool{}
		for _, p := range p.adds {
			pkgs[p] = true
		}
		for _, p := range p.removes {
			if _, ok := pkgs[p]; ok {
				return fmt.Errorf("cannot both add and remove package %s", p)
			}
		}
	}

	return nil
}

func (p *Package) Execute(target *types.Target, tr transport.Transport) (bool, error) {
	pkgPrev, _, _, err := tr.Do([]string{"sha256sum", "/var/lib/dpkg/status"})
	if err != nil {
		return false, fmt.Errorf("error retrieving pre package status: %s", err)
	}
	cmd := []string{
		"apt-get", "-y", "install",
	}
	if p.adds != nil {
		cmd = append(cmd, p.adds...)
	}
	if p.removes != nil {
		for _, p := range p.removes {
			cmd = append(cmd, p+"-")
		}
	}

	stdout, _, res, err := tr.Do(cmd)
	if err != nil {
		return false, fmt.Errorf("executing package changes: %s", err)
	}
	if res != 0 {
		return false, fmt.Errorf("non-zero executing package changes: %d", res)
	}

	for _, l := range strings.Split(string(stdout), "\n") {
		log.Debugf("package:out: %s", l)
	}

	pkgPost, _, _, err := tr.Do([]string{"sha256sum", "/var/lib/dpkg/status"})
	if err != nil {
		return false, fmt.Errorf("error retrieving post package status: %s", err)
	}

	if pkgPrev == nil || (pkgPost != nil && !bytes.Equal(pkgPrev, pkgPost)) {
		return true, nil
	}
	return false, nil
}

func (*Package) Name() string { return "package" }
func (*Package) Always() bool { return false }

var _ Module = (*Package)(nil)
