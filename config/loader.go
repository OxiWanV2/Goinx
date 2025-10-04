package config

import (
    "fmt"
    "io/fs"
    "log"
    "os"
    "path/filepath"

    "github.com/OxiWanV2/Goinx/util"
)

func LoadSitesConfig() ([]SiteConfig, error) {
    enabledDir := "/etc/goinx/sites-enabled"
    availableDir := "/etc/goinx/sites-available"

    var sites []SiteConfig

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

        if !util.Exists(confPath) {
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

        sites = append(sites, conf)
        log.Printf("Chargée config site %s", siteName)
    }

    if len(sites) == 0 {
        log.Println("Aucune config valide trouvée.")
    }

    return sites, nil
}