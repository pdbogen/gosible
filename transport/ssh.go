package transport

import (
	"bytes"
	"fmt"
	"github.com/op/go-logging"
	"github.com/pdbogen/gosible/types"
	"golang.org/x/crypto/ssh"
	"io"
	"strings"
)

type SSH struct {
	Address  string
	Username string
	Port     int32
	Password string
	ssh      *ssh.Client
}

var log = logging.MustGetLogger("gosible/transport/ssh")

func (s *SSH) Do(cmd []string) (stdout []byte, stderr []byte, result int, err error) {
	return s.DoInput(cmd, []byte{})
}

func (s *SSH) DoInput(cmd []string, stdin []byte) (stdout []byte, stderr []byte, result int, err error) {
	return s.DoReader(cmd, bytes.NewBuffer(stdin))
}

func (s *SSH) DoReader(cmd []string, stdin io.Reader) (stdout []byte, stderr []byte, result int, err error) {
	if s == nil {
		return nil, nil, 255, fmt.Errorf("SSH.Do called with nil SSH")
	}

	sess, err := s.ssh.NewSession()
	if err != nil {
		return nil, nil, 255, fmt.Errorf("failed opening session: %s", err)
	}

	outBuf := bytes.Buffer{}
	errBuf := bytes.Buffer{}
	sess.Stdout = &outBuf
	sess.Stderr = &errBuf
	sess.Stdin = stdin

	for i, s := range cmd[1:] {
		cmd[i+1] = escape(s)
	}
	cmdString := cmd[0] + " " + strings.Join(cmd[1:], " ")
	log.Debugf("running: %s", cmdString)
	err = sess.Run(cmdString)

	status := 0
	if err != nil {
		switch err.(type) {
		case *ssh.ExitMissingError:
			break
		case *ssh.ExitError:
			status = err.(*ssh.ExitError).ExitStatus()
			break
		default:
			return nil, nil, 255, fmt.Errorf("running command %s: %s", cmd, err)
		}
	}

	return outBuf.Bytes(), errBuf.Bytes(), status, nil
}

// Connect is "static," in that it does not reference the receive it's called on; so it can be called on a nil pointer
// to SSH. This is good, because it's a sort of constructor to create a live SSH connection.
// Thus, it returns an SSH connection to the given target authenticated with the given credential; or an error, if
// something went wrong.
func (*SSH) Connect(target *types.Target, cred *types.Credential) (Transport, error) {
	if target == nil {
		return nil, fmt.Errorf("SSH.Connect called with nil target")
	}

	if cred == nil {
		return nil, fmt.Errorf("SSH.Connect called with nil credential")
	}

	new_ssh := &SSH{
		Address:  target.Address,
		Port:     target.Port,
		Username: target.User,
		Password: cred.Value,
	}

	hkcb := ssh.InsecureIgnoreHostKey()
	if target.HostKey != nil {
		key, err := ssh.ParsePublicKey([]byte(*target.HostKey))
		if err != nil {
			return nil, fmt.Errorf("provided host key for target %s@%s:%d could not be parsed: %s", target.User, target.Address, target.Port, err)
		}
		hkcb = ssh.FixedHostKey(key)
	}
	config := &ssh.ClientConfig{
		User: target.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(cred.Value),
		},
		HostKeyCallback: hkcb,
	}

	var err error
	new_ssh.ssh, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d", target.Address, target.Port), config)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s@%s:%d: %s", target.User, target.Address, target.Port, err)
	}

	return new_ssh, nil
}

func (s *SSH) Close() {
	if s.ssh != nil {
		s.ssh.Close()
	}
}

// Casting a nil pointer to *Survey and then attempting to assign it as a Module causes a compiler error if Survey does
// not implement Module. This constitutes a contract therefore that Survey *does* implement Module.
var _ Transport = (*SSH)(nil)
