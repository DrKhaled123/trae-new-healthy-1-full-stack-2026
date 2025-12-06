package config

// VPSConfig holds VPS-specific configuration
type VPSConfig struct {
	Host    string `json:"host"`
	Port    int    `json:"port"`
	SSLCert string `json:"ssl_cert"`
	SSLKey  string `json:"ssl_key"`
	Domain  string `json:"domain"`
	Enabled bool   `json:"enabled"`
}

// NewVPSConfig creates a new VPS configuration
func NewVPSConfig() *VPSConfig {
	return &VPSConfig{
		Host:    "0.0.0.0",
		Port:    8080,
		Enabled: false,
	}
}

// IsEnabled returns whether VPS mode is enabled
func (c *VPSConfig) IsEnabled() bool {
	return c.Enabled
}
