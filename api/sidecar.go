package api

import (
	"fmt"
	"strconv"
)

type Sidecar struct {
	CloudSQLProxy *CloudSQLProxySidecar `json:"cloudSqlProxy" yaml:"cloudSqlProxy"`
}

func (s *Sidecar) Valid() error {
	var n int
	if s.CloudSQLProxy != nil {
		n++
		if err := s.CloudSQLProxy.Valid(); err != nil {
			return fmt.Errorf("cloudSqlProxy: %w", err)
		}
	}
	if n != 1 {
		return fmt.Errorf("only 1 sidecar config per item is allowed")
	}
	return nil
}

func (s *Sidecar) Config() *SidecarConfig {
	switch {
	case s.CloudSQLProxy != nil:
		return s.CloudSQLProxy.config()
	}
	return nil
}

type SidecarConfig struct {
	Name      string
	Image     string
	Env       map[string]string
	Command   []string
	Args      []string
	Port      *int
	MountData map[string]string
}

type CloudSQLProxySidecar struct {
	Instance    string `json:"instance" yaml:"instance"`
	Port        int    `json:"port" yaml:"port"`
	Credentials string `json:"credentials" yaml:"credentials"`
}

func (s *CloudSQLProxySidecar) Valid() error {
	if s.Instance == "" {
		return fmt.Errorf("instance is required")
	}
	return nil
}

func (s *CloudSQLProxySidecar) config() *SidecarConfig {
	port := s.Port
	if port <= 0 {
		port = 3300
	}

	cfg := SidecarConfig{
		Name:  "cloudsql-proxy",
		Image: "gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.7.0",
		Port:  &port,
		Args: []string{
			s.Instance,
			"-p=" + strconv.Itoa(port),
			"--max-sigterm-delay=30",
		},
		MountData: map[string]string{},
	}
	if s.Credentials != "" {
		// cfg.Args = append(cfg.Args, "--json-credentials="+s.Credentials)
		cfg.Args = append(cfg.Args, "--credentials-file=/sidecar/cloudsqlproxy/credentials.json")
		cfg.MountData["/sidecar/cloudsqlproxy/credentials.json"] = s.Credentials
	}

	return &cfg
}
