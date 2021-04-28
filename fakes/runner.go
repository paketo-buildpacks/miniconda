package fakes

import "sync"

type Runner struct {
	RunCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			RunPath   string
			LayerPath string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
}

func (f *Runner) Run(param1 string, param2 string) error {
	f.RunCall.Lock()
	defer f.RunCall.Unlock()
	f.RunCall.CallCount++
	f.RunCall.Receives.RunPath = param1
	f.RunCall.Receives.LayerPath = param2
	if f.RunCall.Stub != nil {
		return f.RunCall.Stub(param1, param2)
	}
	return f.RunCall.Returns.Error
}
