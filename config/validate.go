package config

import "fmt"

func ValidateConfigs(sites []SiteConfig) error {
    type portServer struct {
        Port   string
        Server string
    }
    seen := make(map[portServer]bool)

    for _, site := range sites {
        key := portServer{Port: site.Listen, Server: site.ServerName}
        if seen[key] {
            return fmt.Errorf("conflit détecté : domaine %s déjà utilisé sur le port %s", site.ServerName, site.Listen)
        }
        seen[key] = true
    }
    return nil
}