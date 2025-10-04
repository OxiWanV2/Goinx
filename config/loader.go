package config

import (
    "fmt"
    "io/fs"
    "log"
    "os"
    "path/filepath"
)

type SiteWithName struct {
    Name   string
    Config SiteConfig
}

func LoadSitesConfigWithNames() ([]SiteWithName, error) {
    enabledDir := "/etc/goinx/sites-enabled"
    availableDir := "/etc/goinx/sites-available"

    var sites []SiteWithName

    entries, err := os.ReadDir(enabledDir)
    if err != nil {
        return nil, fmt.Errorf("lecture %s impossible : %v", enabledDir, err)
    }

    for _, entry := range entries {
        linkPath := filepath.Join(enabledDir, entry.Name())

        info, err := os.Lstat(linkPath)
        if err != nil {
            log.Printf("Erreur lecture info lien %s : %v", linkPath, err)
            continue
        }
        if info.Mode()&fs.ModeSymlink == 0 {
            log.Printf("Ignoré %s : pas un lien symbolique", linkPath)
            continue
        }

        siteName := entry.Name()
        confPath := filepath.Join(availableDir, siteName, siteName+".conf")

        if _, err := os.Stat(confPath); os.IsNotExist(err) {
            log.Printf("Config manquante %s pour site %s", confPath, siteName)
            continue
        }

        conf, err := ParseConf(confPath)
        if err != nil {
            log.Printf("Erreur parsing %s : %v", confPath, err)
            continue
        }

        if conf.ServerName == "" || conf.Root == "" {
            log.Printf("Config invalide pour site %s (server_name/root manquant)", siteName)
            continue
        }

        sites = append(sites, SiteWithName{
            Name:   siteName,
            Config: conf,
        })
        log.Printf("Chargée config site %s", siteName)
    }

    if len(sites) == 0 {
        log.Println("Aucune config valide trouvée.")
    }

    return sites, nil
}