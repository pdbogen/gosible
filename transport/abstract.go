package transport

import (
	"github.com/pdbogen/gosible/types"
	"io"
)

type TransportConnect func(*types.Target, *types.Credential) (Transport, error)

type Transport interface {
	// Do performs a command via the configured transport, probably on a target host. The first element in cmd
	// should be the binary; the rest will be individual arguments, properly escaped...
	Do(cmd []string) (stdout []byte, stderr []byte, result int, err error)
	DoInput(cmd []string, stdin []byte) (stdout []byte, stderr []byte, result int, err error)
	DoReader(cmd []string, stdin io.Reader) (stdout []byte, stderr []byte, result int, err error)
	Connect(target *types.Target, credential *types.Credential) (Transport, error)
	Close()
}
