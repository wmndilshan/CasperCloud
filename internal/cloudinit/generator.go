package cloudinit

import (
	"bytes"
	"fmt"
	"text/template"
)

type Data struct {
	Hostname     string
	Username     string
	SSHPublicKey string
	Packages     []string
	RunCommands  []string
}

const userDataTemplate = `#cloud-config
hostname: {{ .Hostname }}
users:
  - name: {{ .Username }}
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    ssh_authorized_keys:
      - {{ .SSHPublicKey }}
package_update: true
packages:
{{- range .Packages }}
  - {{ . }}
{{- end }}
runcmd:
{{- range .RunCommands }}
  - {{ . }}
{{- end }}
`

func RenderUserData(in Data) (string, error) {
	if in.Hostname == "" || in.Username == "" || in.SSHPublicKey == "" {
		return "", fmt.Errorf("hostname, username and ssh public key are required")
	}
	tpl, err := template.New("userdata").Parse(userDataTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, in); err != nil {
		return "", err
	}
	return buf.String(), nil
}
