package cloudinit

import (
	"metalnode/pkg/remote"
)

const (
	CloudConfig string = "cloud-config"
	Ignition    string = "ignition"
)

type BootstrapDataParser struct {
	actions []action
}

func NewBootstrapDataParser() *BootstrapDataParser {
	return &BootstrapDataParser{}
}

// Parse the given data into remote.Command to run by ssh
func (p *BootstrapDataParser) Parse(bootstrapData []byte, format []byte) (remote.Command, error) {
	var err error
	// todo format is not currently processed,only support cloud-config
	if string(format) == "" {
		format = []byte(CloudConfig)
	}

	//data, err := base64.StdEncoding.DecodeString(bootstrapData)
	p.actions, err = GetActions(bootstrapData)
	if err != nil {
		return remote.Command{}, nil
	}
	return p.actionToRemoteCmd()
}

// actionToRemoteCmd converts the actions to remote.Command
func (p *BootstrapDataParser) actionToRemoteCmd() (remote.Command, error) {
	var command remote.Command
	for _, action := range p.actions {
		cmds, err := action.Commands()
		if err != nil {
			return command, err
		}
		command.Cmds = append(command.Cmds, cmds...)
	}
	return command, nil
}
