package libvirt

import (
	"bytes"
	"fmt"
	"text/template"
)

type domainXMLInput struct {
	Name        string
	UUID        string
	MemoryKiB   int
	VCPUs       int
	NetworkName string
	DiskPath    string
	SeedISOPath string
}

const domainXMLTemplate = `<domain type='qemu'>
  <name>{{ .Name }}</name>
  <uuid>{{ .UUID }}</uuid>
  <memory unit='KiB'>{{ .MemoryKiB }}</memory>
  <vcpu placement='static'>{{ .VCPUs }}</vcpu>
  <os>
    <type arch='x86_64' machine='pc'>hvm</type>
    <boot dev='hd'/>
    <boot dev='cdrom'/>
  </os>
  <features>
    <acpi/>
    <apic/>
  </features>
  <clock offset='utc'/>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2' discard='unmap'/>
      <source file='{{ .DiskPath }}'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='{{ .SeedISOPath }}'/>
      <target dev='sda' bus='sata'/>
      <readonly/>
    </disk>
    <interface type='network'>
      <source network='{{ .NetworkName }}'/>
      <model type='virtio'/>
    </interface>
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
    <serial type='pty'>
      <target port='0'/>
    </serial>
  </devices>
</domain>
`

func renderDomainXML(in domainXMLInput) (string, error) {
	tpl, err := template.New("domain").Parse(domainXMLTemplate)
	if err != nil {
		return "", fmt.Errorf("parse domain template: %w", err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, in); err != nil {
		return "", fmt.Errorf("execute domain template: %w", err)
	}
	return buf.String(), nil
}
