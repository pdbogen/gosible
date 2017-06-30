package types

// A TaskSet is like a target without all the targeting parameters...
type Set struct {
	Name  string
	Tasks []*Task
}
