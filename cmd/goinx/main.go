package main

import (
    "flag"
    "log"
    "github.com/OxiWanV2/Goinx/config"
)

func main() {
    var cliMode bool
    flag.BoolVar(&cliMode, "cli", false, "Mode console interactif")
    flag.Parse()

    log.Println("Initialisation de Goinx...")

    err := config.SetupGoinx()
    if err != nil {
        log.Fatalf("Erreur lors du setup : %v", err)
    }

    if cliMode {
        config.StartCLI()
        return
    }

    sites, err := config.LoadSitesConfigWithNames()
    if err != nil {
        log.Fatalf("Erreur chargement configuration : %v", err)
    }
    if len(sites) == 0 {
        log.Fatal("Aucun site actif trouvé, arrêt.")
    }

    var configs []config.SiteConfig
    for _, s := range sites {
        configs = append(configs, s.Config)
    }
    err = config.ValidateConfigs(configs)
    if err != nil {
        log.Fatalf("Validation des configurations échouée : %v", err)
    }

    log.Printf("Démarrage des %d sites activés...", len(sites))

    for _, s := range sites {
        err := config.StartServer(s.Name, s.Config)
        if err != nil {
            log.Printf("Erreur démarrage serveur site %s : %v", s.Name, err)
        }
    }

    select {}
}