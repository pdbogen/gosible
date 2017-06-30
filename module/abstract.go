package module

import (
	"github.com/pdbogen/gosible/transport"
	"github.com/pdbogen/gosible/types"
)

type Module interface {
	Name() string
	Execute(*types.Target, transport.Transport) (changed bool, err error)
	Configure(target *types.Target, params map[string]string) error
	Always() bool
}
