package config

import (
    "bufio"
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"
    "time"
	"github.com/OxiWanV2/Goinx/utils"
)

func stopAllServers() {
	activeServersMu.Lock()
	defer activeServersMu.Unlock()

	for name, srv := range activeServers {
		srv.mu.Lock()
		if srv.running {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := srv.httpServer.Shutdown(ctx)
			cancel()
			if err != nil {
				log.Printf("Erreur arrêt serveur %s : %v", name, err)
			} else {
				srv.running = false
				log.Printf("Serveur %s arrêté", name)
			}
		}
		srv.mu.Unlock()
	}
	activeServers = make(map[string]*SiteServer)
}

func StartCLI() {
	sitesConfig, err := LoadSitesConfigWithNames()
	if err != nil {
		fmt.Printf("Erreur chargement sites activés : %v\n", err)
		return
	}

	for _, s := range sitesConfig {
		if err := InitSite(s.Config); err != nil {
			fmt.Printf("Erreur initialisation site %s : %v\n", s.Name, err)
			return
		}
	}

	go StartMainListener()
	go LaunchHttpsServers()

	fmt.Println("Goinx CLI - Commandes: list, enable <site>, disable <site>, testconf <site>, reload, help, exit")

	scanner := bufio.NewScanner(os.Stdin)
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
			fmt.Println("  enable <site>          - active un site (crée lien, valide config, init site)")
			fmt.Println("  disable <site>         - désactive un site (arrête serveur, supprime lien)")
			fmt.Println("  testconf <site>        - teste la config d’un site")
			fmt.Println("  reload                 - recharge la configuration des sites")
			fmt.Println("  exit                   - quitte le CLI")
		case "list":
			handleList()
		case "enable":
			if len(args) < 2 {
				fmt.Println("Usage : enable <nom_site>")
				continue
			}
			siteName := args[1]
			if err := EnableSite(siteName); err != nil {
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
			if err := InitSite(conf); err != nil {
				fmt.Println("Erreur initialisation site :", err)
			} else {
				fmt.Println("Site activé et initialisé :", siteName)
			}
		case "disable":
			if len(args) < 2 {
				fmt.Println("Usage : disable <nom_site>")
				continue
			}
			siteName := args[1]
			if err := StopServer(siteName); err != nil {
				fmt.Println("Erreur arrêt serveur :", err)
			}
			if err := DisableSite(siteName); err != nil {
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
			err := ReloadServers()
			if err != nil {
				fmt.Printf("Erreur reload : %v\n", err)
			} else {
				fmt.Println("Reload terminé.")
			}
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