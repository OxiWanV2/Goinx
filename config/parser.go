package config

import (
    "bufio"
    "os"
    "strings"
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
        }
    }
    if err := scanner.Err(); err != nil {
        return config, err
    }
    return config, nil
}