package config

import (
	"bufio"
	"os"
	"strings"
	"strconv"
)

func ParseConf(path string) (SiteConfig, error) {
	var config SiteConfig

	file, err := os.Open(path)
	if err != nil {
		return config, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		switch parts[0] {
		case "server_name":
			if len(parts) >= 2 {
				config.ServerName = parts[1]
			}
		case "listen":
			if len(parts) >= 2 {
				config.Listen = parts[1]
			}
		case "root":
			if len(parts) >= 2 {
				config.Root = parts[1]
			}
		case "vuejs_rewrite":
			if len(parts) >= 3 {
				config.VuejsRewrite.Path = parts[1]
				config.VuejsRewrite.Fallback = parts[2]
			}
		case "error_pages_dir":
			if len(parts) >= 2 {
				config.ErrorPagesDir = parts[1]
			}
		case "ssl_enabled":
			if len(parts) >= 2 {
				config.SSLEnabled = (parts[1] == "true" || parts[1] == "1")
			}
		case "ssl_cert_file":
			if len(parts) >= 2 {
				config.SSLCertFile = parts[1]
			}
		case "ssl_key_file":
			if len(parts) >= 2 {
				config.SSLKeyFile = parts[1]
			}
		case "use_lets_encrypt":
			if len(parts) >= 2 {
				val := strings.ToLower(parts[1])
				config.UseLetsEncrypt = (val == "true" || val == "1")
			}
		case "backend":
			if len(parts) >= 3 {
				config.BackendRoute = parts[1]
				config.Backend = parts[2]
			}
		case "backend_file":
			if len(parts) >= 2 {
				config.BackendFile = parts[1]
			}
		case "backend_internal_port":
			if len(parts) >= 2 {
				port, err := strconv.Atoi(parts[1])
				if err == nil {
					config.BackendInternalPort = port
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return config, err
	}
	return config, nil
}