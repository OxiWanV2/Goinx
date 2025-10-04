package config

import (
    "bufio"
    "fmt"
    "os"
    "strings"
)

func StartCLI() {
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
            fmt.Println("  enable <site>          - active un site")
            fmt.Println("  disable <site>         - désactive un site")
            fmt.Println("  testconf <site>        - teste la config d’un site")
            fmt.Println("  reload                 - recharge la configuration des sites")
            fmt.Println("  exit                   - quitte le CLI")
        case "list":
            handleList()
        case "enable":
            if len(args) < 2 {
                fmt.Println("Usage : enable <nom_site>")
            } else {
                err := EnableSite(args[1])
                if err != nil {
                    fmt.Println("Erreur :", err)
                } else {
                    fmt.Println("Site activé :", args[1])
                }
            }
        case "disable":
            if len(args) < 2 {
                fmt.Println("Usage : disable <nom_site>")
            } else {
                err := DisableSite(args[1])
                if err != nil {
                    fmt.Println("Erreur :", err)
                } else {
                    fmt.Println("Site désactivé :", args[1])
                }
            }
        case "testconf":
            if len(args) < 2 {
                fmt.Println("Usage : testconf <nom_site>")
            } else {
                handleTestConf(args[1])
            }
        case "reload":
            fmt.Println("Recharge des configurations non implémentée (à faire)")
        case "exit":
            fmt.Println("Sortie.")
            return
        default:
            fmt.Println("Commande inconnue. Tapez 'help'.")
        }
    }
}

func handleList() {
    fmt.Println("Liste des sites activés (à implémenter)")
}

func handleTestConf(siteName string) {
    confPath := fmt.Sprintf("/etc/goinx/sites-available/%s/%s.conf", siteName, siteName)
    conf, err := ParseConf(confPath)
    if err != nil {
        fmt.Printf("Erreur lecture config : %v\n", err)
        return
    }

    fmt.Printf("Config %s testée : %+v\n", siteName, conf)
}