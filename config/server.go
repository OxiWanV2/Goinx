package config

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/acme/autocert"
)

type SiteServer struct {
	Config     SiteConfig
	httpServer *http.Server
	running    bool
	mu         sync.Mutex
}

type Site struct {
	Config  SiteConfig
	Router  *gin.Engine
	Running bool
	Mutex   sync.Mutex
}

var (
	sitesMu         sync.Mutex
	sites           = make(map[string]*Site)
	autocertMgrsMu  sync.Mutex
	autocertMgrs    = make(map[string]*autocert.Manager)
	activeServersMu sync.Mutex
	activeServers   = make(map[string]*SiteServer)
	httpsServerMu   sync.Mutex
	httpsServer     *http.Server
)

func domainPointsToServerIP(domain string) bool {
	domainIPs, err := net.LookupIP(domain)
	if err != nil {
		log.Printf("DNS lookup fail for %s: %v", domain, err)
		return false
	}

	localIPs := map[string]bool{}
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Erreur récupération interfaces réseau: %v", err)
		return false
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil {
				localIPs[ip.String()] = true
			}
		}
	}

	for _, ip := range domainIPs {
		if localIPs[ip.String()] {
			return true
		}
	}

	return false
}

func InitSite(cfg SiteConfig) error {
	r := gin.New()
	r.Use(gin.Recovery())

	r.Static("/", cfg.Root)

	if cfg.VuejsRewrite.Path != "" && cfg.VuejsRewrite.Fallback != "" {
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, cfg.VuejsRewrite.Path) {
				c.File(filepath.Join(cfg.Root, cfg.VuejsRewrite.Fallback))
			} else {
				ServeErrorPage(c, http.StatusNotFound, cfg)
				c.Abort()
			}
		})
	} else {
		r.NoRoute(func(c *gin.Context) {
			ServeErrorPage(c, http.StatusNotFound, cfg)
			c.Abort()
		})
	}

	site := &Site{
		Config: cfg,
		Router: r,
	}

	sitesMu.Lock()
	sites[cfg.ServerName] = site
	sitesMu.Unlock()

	if cfg.UseLetsEncrypt {
		setupLetsEncrypt(site)
	}

	return nil
}

func setupLetsEncrypt(site *Site) {
	host := site.Config.ServerName
	certCacheDir := "/etc/goinx/certs-cache"

	if !domainPointsToServerIP(host) {
		log.Printf("Le domaine %s ne pointe pas vers cette IP. Ignorer Let's Encrypt.", host)
		return
	}

	autocertMgrsMu.Lock()
	m := &autocert.Manager{
		Cache:      autocert.DirCache(certCacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(host),
	}
	autocertMgrs[host] = m
	autocertMgrsMu.Unlock()

	certFile := filepath.Join(certCacheDir, host)
	if fileExists(certFile) {
		log.Printf("Certificat Let's Encrypt pour %s trouvé en cache", host)
	} else {
		log.Printf("Certificat Let's Encrypt pour %s absent, sera généré automatiquement par autocert", host)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func getSiteByHost(host string) *Site {
	sitesMu.Lock()
	defer sitesMu.Unlock()
	if site, exists := sites[host]; exists {
		return site
	}
	if site, ok := sites["default"]; ok {
		return site
	}
	return nil
}

func ReloadServers() error {
	log.Println("Reload des serveurs en cours...")

	sitesConfig, err := LoadSitesConfigWithNames()
	if err != nil {
		return err
	}

	httpsServerMu.Lock()
	if httpsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := httpsServer.Shutdown(ctx)
		cancel()
		if err != nil {
			log.Printf("Erreur arrêt serveur HTTPS : %v", err)
		} else {
			log.Println("Serveur HTTPS arrêté")
		}
		httpsServer = nil
	}
	httpsServerMu.Unlock()

	sitesMu.Lock()
	sites = make(map[string]*Site)
	sitesMu.Unlock()

	autocertMgrsMu.Lock()
	autocertMgrs = make(map[string]*autocert.Manager)
	autocertMgrsMu.Unlock()

	for _, site := range sitesConfig {
		err := InitSite(site.Config)
		if err != nil {
			log.Printf("Erreur initialisation site %s : %v", site.Name, err)
			continue
		}
		log.Printf("Site %s initialisé.", site.Name)
	}

	go LaunchHttpsServers()

	return nil
}

func StartMainListener() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if strings.Contains(host, ":") {
			host = strings.Split(host, ":")[0]
		}

		var site *Site
		if host == "" || net.ParseIP(host) != nil {
			site = getSiteByHost(host)
		} else {
			sitesMu.Lock()
			site, _ = sites[host]
			sitesMu.Unlock()
		}

		if site != nil {
			if site.Config.UseLetsEncrypt {
				target := "https://" + host + r.URL.RequestURI()
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return
			}
			site.Router.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/acme-challenge/" || strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			host := r.Host
			if strings.Contains(host, ":") {
				host = strings.Split(host, ":")[0]
			}
			autocertMgrsMu.Lock()
			m, ok := autocertMgrs[host]
			autocertMgrsMu.Unlock()
			if ok {
				m.HTTPHandler(nil).ServeHTTP(w, r)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:    ":80",
		Handler: finalHandler,
	}

	log.Println("Serveur principal (multi-site) lancé sur port 80")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Serveur principal erreur: %v", err)
	}
}

func LaunchHttpsServers() {
	sitesMu.Lock()
	defer sitesMu.Unlock()

	hasLetsEncrypt := false
	hasManualSSL := false

	for _, site := range sites {
		if site.Config.UseLetsEncrypt {
			hasLetsEncrypt = true
		} else if site.Config.SSLEnabled {
			hasManualSSL = true
		}
	}

	if !hasLetsEncrypt && !hasManualSSL {
		log.Println("Aucun site SSL actif, serveur HTTPS non lancé")
		return
	}

	httpsServerMu.Lock()
	defer httpsServerMu.Unlock()

	if httpsServer != nil {
		log.Println("Serveur HTTPS déjà actif, skip lancement")
		return
	}

	if hasLetsEncrypt {
		hosts := []string{}
		for _, site := range sites {
			if site.Config.UseLetsEncrypt {
				hosts = append(hosts, site.Config.ServerName)
			}
		}
		m := &autocert.Manager{
			Cache:      autocert.DirCache("/etc/goinx/certs-cache"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(hosts...),
		}

		tlsConfig := &tls.Config{
			GetCertificate: m.GetCertificate,
			NextProtos:     []string{"h2", "http/1.1", "acme-tls/1"},
		}

		autocertMgrsMu.Lock()
		for _, host := range hosts {
			autocertMgrs[host] = m
		}
		autocertMgrsMu.Unlock()

		httpsServer = &http.Server{
			Addr:      ":443",
			TLSConfig: tlsConfig,
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				host := r.Host
				if strings.Contains(host, ":") {
					host = strings.Split(host, ":")[0]
				}
				sitesMu.Lock()
				site, exists := sites[host]
				sitesMu.Unlock()
				if exists {
					site.Router.ServeHTTP(w, r)
				} else {
					http.NotFound(w, r)
				}
			}),
		}

		go func() {
			log.Println("Serveur HTTPS auto (Let's Encrypt) lancé sur port 443")
			if err := httpsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Printf("Erreur serveur HTTPS auto : %v", err)
			}
		}()
		return
	}

	for _, site := range sites {
		if site.Config.SSLEnabled && !site.Config.UseLetsEncrypt {
			if !fileExists(site.Config.SSLCertFile) || !fileExists(site.Config.SSLKeyFile) {
				log.Printf("Certificat ou clé ssl introuvable pour site %s", site.Config.ServerName)
				continue
			}
			listenAddr := ":" + site.Config.Listen
			srv := &http.Server{
				Addr:    listenAddr,
				Handler: site.Router,
				TLSConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			}
			go func(srv *http.Server, siteName, certFile, keyFile string) {
				log.Printf("Serveur SSL manuel pour %s sur port %s", siteName, listenAddr)
				if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
					log.Printf("Erreur SSL manuel %s : %v", siteName, err)
				}
			}(srv, site.Config.ServerName, site.Config.SSLCertFile, site.Config.SSLKeyFile)
		}
	}
}

func StopServer(siteName string) error {
	activeServersMu.Lock()
	siteSrv, exists := activeServers[siteName]
	activeServersMu.Unlock()
	if !exists || !siteSrv.running {
		return nil
	}
	siteSrv.mu.Lock()
	defer siteSrv.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := siteSrv.httpServer.Shutdown(ctx)
	if err != nil {
		return err
	}
	siteSrv.running = false
	activeServersMu.Lock()
	delete(activeServers, siteName)
	activeServersMu.Unlock()
	log.Printf("Serveur site %s arrêté proprement", siteName)
	return nil
}

func IsServerRunning(siteName string) bool {
	activeServersMu.Lock()
	defer activeServersMu.Unlock()
	s, ok := activeServers[siteName]
	return ok && s.running
}