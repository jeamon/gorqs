package goq

type Runner interface {
	Run() error
}

type Jobber interface {
	GetID() int64
	Runner
}

type Job struct {
	id int64
	r  Runner
}

func (j *Job) GetID() int64 {
	return j.id
}

func (j *Job) Run() error {
	return j.r.Run()
}
