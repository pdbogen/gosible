package types

// This YAML parsing is slightly fancy; a Task is an object. Its yaml `name` field becomes Name.
// Every other field becomes a key in Modules.
type Task struct {
	Name    string
	Modules map[string]map[string]string `yaml:",inline"`
}
