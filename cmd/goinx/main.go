package main

import (
    "log"
    "github.com/OxiWanV2/Goinx/setup"
)

func main() {
    log.Println("Initialisation de Goinx...")

    err := setup.SetupGoinx()
    if err != nil {
        log.Fatalf("Erreur setup : %v", err)
    }

    log.Println("Setup terminé, démarrage serveur...")
}