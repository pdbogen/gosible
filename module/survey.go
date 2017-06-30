package module

import (
	"errors"
	"fmt"
	"github.com/op/go-logging"
	"github.com/pdbogen/gosible/transport"
	"github.com/pdbogen/gosible/types"
	"gopkg.in/yaml.v2"
)

var log = logging.MustGetLogger("gosible/module/survey")

type Survey struct {
}

func (*Survey) Configure(*types.Target, map[string]string) error { return nil }

func (s *Survey) Execute(target *types.Target, tr transport.Transport) (bool, error) {
	if target == nil {
		return false, errors.New("Survey.Execute received nil target")
	}
	if tr == nil {
		return false, errors.New("Survey.Execute received nil transport")
	}

	out, _, res, err := tr.Do([]string{"hostname"})
	if err != nil {
		return false, fmt.Errorf("collecting hostname: %s", err)
	}
	if res != 0 {
		return false, fmt.Errorf("non-zero return collecting hostname: %d", res)
	}
	if target.Metadata == nil {
		target.Metadata = map[string]string{}
	}

	target.Metadata["hostname"] = string(out)

	yaml, _ := yaml.Marshal(target.Metadata)
	log.Debugf("survey results for %s: %s", target.Name, string(yaml))
	return false, nil
}

func (*Survey) Always() bool {
	return true
}

func (*Survey) Name() string {
	return "survey"
}

// Casting a nil pointer to *Survey and then attempting to assign it as a Module causes a compiler error if Survey does
// not implement Module. This constitutes a contract therefore that Survey *does* implement Module.
var _ Module = (*Survey)(nil)
