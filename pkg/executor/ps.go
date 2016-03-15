package executor

import (
	ps "github.com/mitchellh/go-ps"
	"os"
)

type Process struct {
	PPid       int         `json:"ppid"`
	Pid        int         `json:"pid"`
	Cmd        string      `json:"cmd"`
	Usage      string      `json:"usage,omitempty"`
	SystemTime string      `json:"system_time,omitempty"`
	UserTime   string      `json:"user_time,omitempty"`
	Process    *os.Process `json:"-"`
}

func children_processes() ([]Process, error) {
	children := []Process{}

	myPid := os.Getpid()
	pss, err := ps.Processes()
	if err != nil {
		return nil, err
	}
	for _, p := range pss {
		if p.PPid() == myPid {
			cp, err := os.FindProcess(p.Pid())
			if err != nil {
				return nil, err
			}
			children = append(children, Process{
				PPid:    p.PPid(),
				Pid:     p.Pid(),
				Cmd:     p.Executable(),
				Process: cp,
			})
		}
	}
	return children, nil
}
