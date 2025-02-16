package container

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"proxmox-lxc-compose/pkg/common"
	"proxmox-lxc-compose/pkg/logging"
)

// // VPNConfig represents OpenVPN configuration
// type VPNConfig struct {
// 	Remote   string            // VPN server address
// 	Port     int               // VPN server port
// 	Protocol string            // udp or tcp
// 	Config   string            // Path to OpenVPN config file
// 	Auth     map[string]string // Authentication credentials
// 	CA       string            // CA certificate content
// 	Cert     string            // Client certificate content
// 	Key      string            // Client key content
// }

const vpnConfigTemplate = `client
dev tun
proto {{ .Protocol }}
remote {{ .Remote }} {{ .Port }}
resolv-retry infinite
nobind
persist-key
persist-tun
verb 3

{{ if .CA }}
<ca>
{{ .CA }}
</ca>
{{ end }}

{{ if and .Cert .Key }}
<cert>
{{ .Cert }}
</cert>
<key>
{{ .Key }}
</key>
{{ end }}

{{ if .Auth }}
auth-user-pass
{{ end }}
`

// ConfigureVPN sets up VPN for a container
func (m *LXCManager) ConfigureVPN(name string, vpn *common.VPNConfig) error {
	if vpn == nil {
		return nil
	}

	// Check if container exists
	if !m.ContainerExists(name) {
		return fmt.Errorf("container %s does not exist", name)
	}

	// Validate required fields
	if vpn.Remote == "" {
		return fmt.Errorf("VPN remote address is required")
	}
	if vpn.Port == 0 {
		return fmt.Errorf("VPN port is required")
	}
	if vpn.Protocol == "" {
		return fmt.Errorf("VPN protocol is required")
	}
	if vpn.Protocol != "udp" && vpn.Protocol != "tcp" {
		return fmt.Errorf("invalid VPN protocol: %s (must be 'udp' or 'tcp')", vpn.Protocol)
	}

	logging.Debug("Configuring VPN", "container", name)

	// Create VPN config directory in container
	vpnDir := filepath.Join(m.configPath, name, "vpn")
	if err := os.MkdirAll(vpnDir, 0755); err != nil {
		return fmt.Errorf("failed to create VPN config directory: %w", err)
	}

	// Write OpenVPN config
	tmpl, err := template.New("vpn").Parse(vpnConfigTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse VPN template: %w", err)
	}

	configPath := filepath.Join(vpnDir, "client.conf")
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create VPN config: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, vpn); err != nil {
		return fmt.Errorf("failed to write VPN config: %w", err)
	}

	// Write CA certificate if provided
	if vpn.CA != "" {
		caPath := filepath.Join(vpnDir, "ca.crt")
		if err := os.WriteFile(caPath, []byte(vpn.CA), 0644); err != nil {
			return fmt.Errorf("failed to write CA certificate: %w", err)
		}
	}

	// Write auth credentials if provided
	if len(vpn.Auth) > 0 {
		authPath := filepath.Join(vpnDir, "auth.conf")
		content := fmt.Sprintf("%s\n%s\n", vpn.Auth["username"], vpn.Auth["password"])
		if err := os.WriteFile(authPath, []byte(content), 0600); err != nil {
			return fmt.Errorf("failed to write VPN credentials: %w", err)
		}
	}

	// Add OpenVPN service to container startup
	configPath = filepath.Join(m.configPath, name, "config")
	f, err = os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open container config: %w", err)
	}
	defer f.Close()

	lines := []string{
		"lxc.hook.pre-start = openvpn --daemon --config /etc/openvpn/client.conf",
		"lxc.hook.post-stop = pkill openvpn",
	}

	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write container config: %w", err)
		}
	}

	logging.Debug("VPN configured successfully", "container", name)
	return nil
}

// RemoveVPN removes VPN configuration from a container
func (m *LXCManager) RemoveVPN(name string) error {
	vpnDir := filepath.Join(m.configPath, name, "vpn")
	if err := os.RemoveAll(vpnDir); err != nil {
		return fmt.Errorf("failed to remove VPN directory: %w", err)
	}
	return nil
}
