package cloudinit

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"github.com/git-czy/cluster-api-metalnode/pkg/remote"
	"github.com/pkg/errors"
	"io"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

type writeFilesAction struct {
	Files []files `json:"write_files,"`
}

type files struct {
	Path        string `json:"path,"`
	Encoding    string `json:"encoding,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Permissions string `json:"permissions,omitempty"`
	Content     string `json:"content,"`
	Append      bool   `json:"append,"`
}

func (a *writeFilesAction) Unmarshal(data []byte) error {
	return yaml.Unmarshal(data, a)
}

// Commands return a remote.Commands to run by ssh
func (a *writeFilesAction) Commands() (remote.Commands, error) {
	var cmds remote.Commands
	for _, f := range a.Files {
		path := fixPath(f.Path)
		encodings := fixEncoding(f.Encoding)
		owner := fixOwner(f.Owner)
		permissions := fixPermissions(f.Permissions)
		content, err := fixContent(f.Content, encodings)
		if err != nil {
			return nil, err
		}
		// 创建文件目录
		cmds = append(cmds, joinMkdirCmd(path))
		// 写入文件
		cmds = append(cmds, echoContentCmd(content, path, f.Append))
		// 设置权限
		if permissions != "0644" {
			cmds = append(cmds, joinPermissionsCmd(permissions, path))
		}
		// 设置所有者
		if owner != "root:root" {
			cmds = append(cmds, joinOwnerCmd(owner, path))
		}
	}
	return cmds, nil
}

func joinMkdirCmd(path string) string {
	return "sudo mkdir -p " + filepath.Dir(path)
}

func echoContentCmd(content string, path string, append bool) string {
	if append {
		return "echo '" + content + "' | sudo tee -a " + path
	}
	return "echo '" + content + "' | sudo tee " + path
}

func joinPermissionsCmd(permissions string, path string) string {
	return "sudo chmod " + permissions + " " + path
}

func joinOwnerCmd(owner string, path string) string {
	return "sudo chown " + owner + " " + path
}

func fixPath(p string) string {
	return strings.TrimSpace(p)
}

func fixOwner(o string) string {
	o = strings.TrimSpace(o)
	if o != "" {
		return o
	}
	return "root:root"
}

func fixPermissions(p string) string {
	p = strings.TrimSpace(p)
	if p != "" {
		return p
	}
	return "0644"
}

func fixEncoding(e string) []string {
	e = strings.ToLower(e)
	e = strings.TrimSpace(e)

	switch e {
	case "gz", "gzip":
		return []string{"application/x-gzip"}
	case "gz+base64", "gzip+base64", "gz+b64", "gzip+b64":
		return []string{"application/base64", "application/x-gzip"}
	case "base64", "b64":
		return []string{"application/base64"}
	}

	return []string{"text/plain"}
}

func fixContent(content string, encodings []string) (string, error) {
	for _, e := range encodings {
		switch e {
		case "application/base64":
			rByte, err := base64.StdEncoding.DecodeString(content)
			if err != nil {
				return content, errors.WithStack(err)
			}
			return string(rByte), nil
		case "application/x-gzip":
			rByte, err := gUnzipData([]byte(content))
			if err != nil {
				return content, err
			}
			return string(rByte), nil
		case "text/plain":
			return content, nil
		default:
			return content, errors.Errorf("Unknown bootstrap data encoding: %q", content)
		}
	}
	return content, nil
}

func gUnzipData(data []byte) ([]byte, error) {
	var r io.Reader
	var err error
	b := bytes.NewBuffer(data)
	r, err = gzip.NewReader(b)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var resB bytes.Buffer
	_, err = resB.ReadFrom(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return resB.Bytes(), nil
}
