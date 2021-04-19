package worker

import "encoding/json"

// Args are the arguments passed into a job
type Args map[string]interface{}

func (a Args) String() string {
	b, _ := json.Marshal(a)
	return string(b)
}

// Job to be processed by a Worker
type Job struct {
	Handler  string
	Exchange string
	Args     Args
}

func (j Job) String() string {
	b, _ := json.Marshal(j)
	return string(b)
}
