package cloudinit

import (
	"fmt"
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
	cmds := remote.Commands{}
	for _, cmd := range a.Cmds {
		cmds = append(cmds, fmt.Sprintf("sudo %s", cmd))
	}
	return cmds, nil
}
