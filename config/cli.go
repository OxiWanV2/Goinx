package config

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"github.com/OxiWanV2/Goinx/backend"
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

	fmt.Println("Goinx CLI - Commandes: list, enable <site>, disable <site>, testconf <site>, reload, log <site>, help, exit")

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
			fmt.Println("  enable <site>          - active un site (crée lien et initialise frontend+backend)")
			fmt.Println("  disable <site>         - désactive un site (arrête serveur + backend, supprime lien)")
			fmt.Println("  testconf <site>        - teste la config d’un site")
			fmt.Println("  reload                 - recharge la configuration des sites et relance tous serveurs")
			fmt.Println("  log <site>             - affiche les logs en temps réel du backend du site")
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
				fmt.Println("Erreur activer site :", err)
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

			if err := backend.StopBackend(siteName); err != nil {
				fmt.Println("Erreur arrêt backend :", err)
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

		case "log":
			if len(args) < 2 {
				fmt.Println("Usage : log <nom_site>")
				continue
			}
			siteName := args[1]

			enabledPath := filepath.Join("/etc/goinx/sites-enabled", siteName)
			if !util.LinkExists(enabledPath) {
				fmt.Printf("Site \"%s\" non trouvé ou non activé.\n", siteName)
				continue
			}

			logChan, exists := backend.GetBackendLogChannel(siteName)
			if !exists {
				fmt.Printf("Aucun backend en cours ou pas de logs disponibles pour le site \"%s\".\n", siteName)

				activeBackends := backend.GetActiveBackends()
				if len(activeBackends) == 0 {
					fmt.Println("Aucun backend actif actuellement.")
				} else {
					fmt.Println("Backends actifs :")
					for _, s := range activeBackends {
						fmt.Printf(" - %s\n", s)
					}
				}
				continue
			}

			fmt.Printf("Affichage des logs en temps réel pour backend du site %s (Ctrl+C pour quitter)\n", siteName)

			done := make(chan os.Signal, 1)
			signal.Notify(done, os.Interrupt, syscall.SIGTERM)

		loop:
			for {
				select {
				case line, ok := <-logChan:
					if !ok {
						fmt.Println("Fin des logs.")
						break loop
					}
					fmt.Println(line)
				case <-done:
					fmt.Println("\nInterruption reçue, arrêt affichage logs.")
					break loop
				}
			}

		case "exit":
			fmt.Println("Sortie.")
			stopAllServers()
			return

		default:
			fmt.Println("Commande inconnue. Tapez 'help' pour la liste des commandes.")
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