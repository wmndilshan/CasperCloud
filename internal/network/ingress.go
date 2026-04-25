package network

import (
	"bytes"
	"errors"
	"fmt"
	"net/netip"
	"os/exec"
	"strings"
)

// Ingress applies host-level DNAT/SNAT for floating IPs using iptables, scoped by strict
// destination/source matches and a unique comment so libvirt virbr0 rules are not affected.
type Ingress struct {
	iptablesPath       string
	privateExcludeCIDR string // traffic to this CIDR is not SNATed to the public floating IP
}

func NewIngress(iptablesPath, privateExcludeCIDR string) *Ingress {
	if iptablesPath == "" {
		iptablesPath = "iptables"
	}
	if privateExcludeCIDR == "" {
		privateExcludeCIDR = "192.168.122.0/24"
	}
	return &Ingress{iptablesPath: iptablesPath, privateExcludeCIDR: privateExcludeCIDR}
}

func normalizeHost(ip string) string {
	ip = strings.TrimSpace(ip)
	if strings.Contains(ip, "/") {
		if p, err := netip.ParsePrefix(ip); err == nil {
			return p.Addr().String()
		}
	}
	if a, err := netip.ParseAddr(ip); err == nil {
		return a.String()
	}
	return ip
}

func ruleComment(publicHost, privateHost string) string {
	return fmt.Sprintf("caspercloud-fip-%s-%s", publicHost, privateHost)
}

func (n *Ingress) preroutingDNATArgs(publicHost, privateHost, comment string, add bool) []string {
	op := "-A"
	if !add {
		op = "-D"
	}
	pub := publicHost + "/32"
	return []string{"-t", "nat", op, "PREROUTING",
		"-d", pub,
		"-m", "comment", "--comment", comment,
		"-j", "DNAT", "--to-destination", privateHost,
	}
}

func (n *Ingress) postroutingSNATArgs(publicHost, privateHost, comment string, add bool) []string {
	op := "-A"
	if !add {
		op = "-D"
	}
	priv := privateHost + "/32"
	return []string{"-t", "nat", op, "POSTROUTING",
		"-s", priv,
		"!", "-d", n.privateExcludeCIDR,
		"-m", "comment", "--comment", comment,
		"-j", "SNAT", "--to-source", publicHost,
	}
}

func (n *Ingress) iptablesCheck(args []string) (exists bool, err error) {
	cmd := exec.Command(n.iptablesPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err == nil {
		return true, nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("%s %v: %w (%s)", n.iptablesPath, args, err, strings.TrimSpace(stderr.String()))
}

func (n *Ingress) iptablesRun(args []string) error {
	cmd := exec.Command(n.iptablesPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w (%s)", n.iptablesPath, args, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func argsWithCheckOp(args []string) []string {
	out := append([]string(nil), args...)
	for i := range out {
		if out[i] == "-A" || out[i] == "-D" {
			out[i] = "-C"
			return out
		}
	}
	return out
}

func (n *Ingress) iptablesRunDeleteIgnoreMissing(args []string) error {
	cmd := exec.Command(n.iptablesPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return nil
		}
		return fmt.Errorf("%s %v: %w (%s)", n.iptablesPath, args, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// AssociateIP adds PREROUTING DNAT (public → private) and POSTROUTING SNAT (private → public for non-local dest).
func (n *Ingress) AssociateIP(publicIP, privateIP string) error {
	pub := normalizeHost(publicIP)
	priv := normalizeHost(privateIP)
	if pub == "" || priv == "" {
		return fmt.Errorf("associate: empty public or private ip")
	}
	comment := ruleComment(pub, priv)

	prCheck := argsWithCheckOp(n.preroutingDNATArgs(pub, priv, comment, true))
	okPR, err := n.iptablesCheck(prCheck)
	if err != nil {
		return err
	}
	poCheck := argsWithCheckOp(n.postroutingSNATArgs(pub, priv, comment, true))
	okPO, err := n.iptablesCheck(poCheck)
	if err != nil {
		return err
	}
	if !okPR {
		if err := n.iptablesRun(n.preroutingDNATArgs(pub, priv, comment, true)); err != nil {
			return err
		}
	}
	if !okPO {
		if err := n.iptablesRun(n.postroutingSNATArgs(pub, priv, comment, true)); err != nil {
			return err
		}
	}
	return nil
}

// DisassociateIP removes the specific DNAT/SNAT rules created for this pair.
func (n *Ingress) DisassociateIP(publicIP, privateIP string) error {
	pub := normalizeHost(publicIP)
	priv := normalizeHost(privateIP)
	if pub == "" || priv == "" {
		return fmt.Errorf("disassociate: empty public or private ip")
	}
	comment := ruleComment(pub, priv)

	_ = n.iptablesRunDeleteIgnoreMissing(n.preroutingDNATArgs(pub, priv, comment, false))
	_ = n.iptablesRunDeleteIgnoreMissing(n.postroutingSNATArgs(pub, priv, comment, false))
	return nil
}

// FloatingNATRulesPresent returns true if both PREROUTING DNAT and POSTROUTING SNAT rules exist.
func (n *Ingress) FloatingNATRulesPresent(publicIP, privateIP string) (bool, error) {
	pub := normalizeHost(publicIP)
	priv := normalizeHost(privateIP)
	comment := ruleComment(pub, priv)

	prCheck := argsWithCheckOp(n.preroutingDNATArgs(pub, priv, comment, true))
	okPR, err := n.iptablesCheck(prCheck)
	if err != nil {
		return false, err
	}
	poCheck := argsWithCheckOp(n.postroutingSNATArgs(pub, priv, comment, true))
	okPO, err := n.iptablesCheck(poCheck)
	if err != nil {
		return false, err
	}
	return okPR && okPO, nil
}
