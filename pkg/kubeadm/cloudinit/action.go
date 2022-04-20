package cloudinit

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/git-czy/cluster-api-metalnode/pkg/remote"
	"regexp"
	"strings"
)

type action interface {
	Unmarshal(userData []byte) error
	Commands() (remote.Commands, error)
}

const (
	writefiles = "write_files"
	runcmd     = "runcmd"
)

func getActionByName(name string) (action, error) {
	switch name {
	case writefiles:
		return &writeFilesAction{}, nil
	case runcmd:
		return &runCmdAction{}, nil
	default:
		return nil, fmt.Errorf("unknown action %q", name)
	}
}

func GetActions(data []byte) ([]action, error) {
	actionRegEx := regexp.MustCompile(`^[a-zA-Z_]*:`)
	lines := make([]string, 0)
	actions := make([]action, 0)
	var act action

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		// if the line is key/top level action
		if actionRegEx.MatchString(line) {
			if act != nil {
				actionBlock := strings.Join(lines, "\n")
				if err := act.Unmarshal([]byte(actionBlock)); err != nil {
					fmt.Println(err.Error())
				}
				actions = append(actions, act)
				lines = lines[:0]
			}
			actionName := strings.TrimSuffix(line, ":")
			act, _ = getActionByName(actionName)
		}
		lines = append(lines, line)
	}

	if act != nil {
		actionBlock := strings.Join(lines, "\n")
		if err := act.Unmarshal([]byte(actionBlock)); err != nil {
			fmt.Println(err.Error())
		}
		actions = append(actions, act)
	}

	return actions, nil
}
