package types

type Target struct {
	Name           string
	Address        string
	Port           int32
	User           string
	CredentialName string `yaml:"credentialName"`
	TransportName  string // Default: SSH
	Metadata       map[string]string
	Tasks          []*Task
	HostKey        *string
}
