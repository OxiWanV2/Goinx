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

	sitesConfig, err := LoadSitesConfigWithNames()
	if err != nil {
		fmt.Printf("Erreur chargement des sites activés : %v\n", err)
		return
	}

	for _, s := range sitesConfig {
		conf := s.Config

		err := ValidateConfigs([]SiteConfig{conf})
		if err != nil {
			fmt.Printf("Config invalide %s : %v\n", s.Name, err)
			continue
		}

		sitesMu.Lock()
		site, exists := sites[conf.ServerName]
		sitesMu.Unlock()

		if exists {
			site.Mutex.Lock()
			site.Router = nil
			site.Mutex.Unlock()
		}

		err = InitSite(conf)
		if err != nil {
			fmt.Printf("Erreur reload site %s : %v\n", s.Name, err)
			continue
		}

		fmt.Printf("Site %s reloadé avec succès.\n", s.Name)
	}
}

func StartCLI() {
	sitesConfig, err := LoadSitesConfigWithNames()
	if err != nil {
		fmt.Printf("Erreur chargement sites activés : %v\n", err)
		return
	}

	for _, s := range sitesConfig {
		err := InitSite(s.Config)
		if err != nil {
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
			err = InitSite(conf)
			if err != nil {
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