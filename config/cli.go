package config

import (
    "bufio"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "github.com/OxiWanV2/Goinx/utils"
)

func reloadSites() {
    fmt.Println("Reload des sites activés...")

    activeServersMu.Lock()
    activeNames := make([]string, 0, len(activeServers))
    for name := range activeServers {
        activeNames = append(activeNames, name)
    }
    activeServersMu.Unlock()

    for _, siteName := range activeNames {
        confPath := filepath.Join("/etc/goinx/sites-available", siteName, siteName+".conf")

        conf, err := ParseConf(confPath)
        if err != nil {
            fmt.Printf("Erreur lecture config %s : %v\n", siteName, err)
            continue
        }

        err = ValidateConfigs([]SiteConfig{conf})
        if err != nil {
            fmt.Printf("Config invalide %s : %v\n", siteName, err)
            continue
        }

        fmt.Printf("Redémarrage du site %s...\n", siteName)
        err = StopServer(siteName)
        if err != nil {
            fmt.Printf("Erreur arrêt ancienne instance %s : %v\n", siteName, err)
            continue
        }
        err = StartServer(siteName, conf)
        if err != nil {
            fmt.Printf("Erreur démarrage serveur %s : %v\n", siteName, err)
            continue
        }
        fmt.Printf("Site %s reloadé avec succès.\n", siteName)
    }
}

func StartCLI() {
    sites, err := LoadSitesConfigWithNames()
    if err != nil {
        fmt.Printf("Erreur chargement sites activés : %v\n", err)
    } else {
        var configs []SiteConfig
        for _, s := range sites {
            configs = append(configs, s.Config)
        }
        err = ValidateConfigs(configs)
        if err != nil {
            fmt.Printf("Erreur validation des configs : %v\n", err)
        } else {
            for _, site := range sites {
                err := StartServer(site.Name, site.Config)
                if err != nil {
                    fmt.Printf("Erreur démarrage serveur site %s : %v\n", site.Name, err)
                } else {
                    fmt.Printf("Serveur site %s démarré automatiquement.\n", site.Name)
                }
            }
        }
    }

    scanner := bufio.NewScanner(os.Stdin)
    fmt.Println("Goinx CLI - Commandes: list, enable <site>, disable <site>, testconf <site>, reload, help, exit")

    for {
        fmt.Print("> ")
        if !scanner.Scan() {
            break
        }
        line := strings.TrimSpace(scanner.Text())
        if line == "" {
            continue
        }
        args := strings.Fields(line)
        cmd := args[0]

        switch cmd {
        case "help":
            fmt.Println("Commandes disponibles :")
            fmt.Println("  list                   - liste les sites disponibles et leur état")
            fmt.Println("  enable <site>          - active un site (crée lien, valide config, démarre serveur)")
            fmt.Println("  disable <site>         - désactive un site (arrête serveur, supprime lien)")
            fmt.Println("  testconf <site>        - teste la config d’un site")
            fmt.Println("  reload                 - recharge la configuration des sites (non encore implémenté)")
            fmt.Println("  exit                   - quitte le CLI")
        case "list":
            handleList()
        case "enable":
            if len(args) < 2 {
                fmt.Println("Usage : enable <nom_site>")
                continue
            }
            siteName := args[1]

            err := EnableSite(siteName)
            if err != nil {
                fmt.Println("Erreur :", err)
                continue
            }

            confPath := filepath.Join("/etc/goinx/sites-available", siteName, siteName+".conf")
            conf, err := ParseConf(confPath)
            if err != nil {
                fmt.Println("Erreur lecture config :", err)
                continue
            }
            if err := ValidateConfigs([]SiteConfig{conf}); err != nil {
                fmt.Println("Config invalide :", err)
                continue
            }

            err = StartServer(siteName, conf)
            if err != nil {
                fmt.Println("Erreur démarrage serveur :", err)
            } else {
                fmt.Println("Site activé et serveur démarré :", siteName)
            }
        case "disable":
            if len(args) < 2 {
                fmt.Println("Usage : disable <nom_site>")
                continue
            }
            siteName := args[1]

            err := StopServer(siteName)
            if err != nil {
                fmt.Println("Erreur arrêt serveur :", err)
                continue
            }

            err = DisableSite(siteName)
            if err != nil {
                fmt.Println("Erreur désactivation site :", err)
            } else {
                fmt.Println("Site désactivé et serveur arrêté :", siteName)
            }
        case "testconf":
            if len(args) < 2 {
                fmt.Println("Usage : testconf <nom_site>")
                continue
            }
            siteName := args[1]
            confPath := filepath.Join("/etc/goinx/sites-available", siteName, siteName+".conf")
            conf, err := ParseConf(confPath)
            if err != nil {
                fmt.Printf("Erreur lecture config : %v\n", err)
                continue
            }
            fmt.Printf("Config %s testée : %+v\n", siteName, conf)
        case "reload":
            reloadSites()
        case "exit":
            fmt.Println("Sortie.")
            return
        default:
            fmt.Println("Commande inconnue. Tapez 'help'.")
        }
    }
}

func handleList() {
    availableDir := "/etc/goinx/sites-available"
    enabledDir := "/etc/goinx/sites-enabled"

    sitesAvailable, err := os.ReadDir(availableDir)
    if err != nil {
        fmt.Printf("Erreur lecture %s : %v\n", availableDir, err)
        return
    }

    fmt.Println("Liste des sites disponibles :")
    for _, site := range sitesAvailable {
        siteName := site.Name()
        if !site.IsDir() {
            continue
        }
        enabledPath := filepath.Join(enabledDir, siteName)
        state := "Désactivé"
        if util.LinkExists(enabledPath) {
            if IsServerRunning(siteName) {
                state = "Activé (serveur en cours)"
            } else {
                state = "Activé (serveur arrêté)"
            }
        }
        fmt.Printf("  - %s : %s\n", siteName, state)
    }
}