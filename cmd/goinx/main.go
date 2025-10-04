package main

import (
    "flag"
    "log"
    "github.com/OxiWanV2/Goinx/config"
    "github.com/OxiWanV2/Goinx/util"
    "github.com/gin-gonic/gin"
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

    sites, err := config.LoadSitesConfig()
    if err != nil {
        log.Fatalf("Erreur chargement configuration : %v", err)
    }
    if len(sites) == 0 {
        log.Fatal("Aucun site actif trouvé, arrêt.")
    }

    err = config.ValidateConfigs(sites)
    if err != nil {
        log.Fatalf("Validation des configurations échouée : %v", err)
    }

    log.Printf("Démarrage des sites (%d)...", len(sites))

    firstSite := sites[0]
    log.Printf("Lancement site %s sur port %s, racine %s", firstSite.ServerName, firstSite.Listen, firstSite.Root)

    gin.SetMode(gin.ReleaseMode)
    r := gin.Default()

    r.Static("/", firstSite.Root)

    r.NoRoute(func(c *gin.Context) {
        c.File(firstSite.Root + "/" + firstSite.VuejsRewrite.Fallback)
    })

    err = r.Run(":" + firstSite.Listen)
    if err != nil {
        log.Fatalf("Erreur lancement serveur : %v", err)
    }
}