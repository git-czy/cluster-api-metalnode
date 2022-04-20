package cloudinit

import (
	"github.com/git-czy/cluster-api-metalnode/pkg/remote"
	"sigs.k8s.io/yaml"
)

type runCmdAction struct {
	Cmds remote.Commands `json:"runcmd"`
}

func (a *runCmdAction) Unmarshal(data []byte) error {
	return yaml.Unmarshal(data, a)
}

func (a *runCmdAction) Commands() (remote.Commands, error) {
	return a.Cmds, nil
}
