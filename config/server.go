package config

import (
	"context"
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
	sitesMu        sync.Mutex
	sites          = make(map[string]*Site)
	autocertMgrsMu sync.Mutex
	autocertMgrs   = make(map[string]*autocert.Manager)
	activeServersMu sync.Mutex
	activeServers   = make(map[string]*SiteServer)
)

func domainPointsToServerIP(domain string) bool {
	domainIPs, err := net.LookupIP(domain)
	if err != nil {
		log.Printf("DNS lookup fail for %s: %v", domain, err)
		return false
	}

	localIPs := make(map[string]bool)
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Erreur récupération interfaces réseau: %v", err)
		return false
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
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
	certFile := filepath.Join(certCacheDir, host)

	hasCert := fileExists(certFile)

	if !domainPointsToServerIP(host) {
		log.Printf("Le domaine %s ne pointe pas vers cette IP. Ignorer Let's Encrypt.", host)
		return
	}

	if !hasCert {
		log.Printf("Certificat Let's Encrypt pour %s absent, démarrage autocert", host)
	} else {
		log.Printf("Certificat Let's Encrypt pour %s trouvé en cache", host)
	}

	autocertMgrsMu.Lock()
	m := &autocert.Manager{
		Cache:      autocert.DirCache(certCacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(host),
	}
	autocertMgrs[host] = m
	autocertMgrsMu.Unlock()
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

	for _, site := range sites {
		cfg := site.Config
		if cfg.UseLetsEncrypt {
			autocertMgrsMu.Lock()
			m, ok := autocertMgrs[cfg.ServerName]
			autocertMgrsMu.Unlock()
			if !ok {
				log.Printf("Autocert manager pour %s introuvable, serveur TLS non démarré", cfg.ServerName)
				continue
			}
			tlsSrv := &http.Server{
				Addr:      ":443",
				TLSConfig: m.TLSConfig(),
				Handler:   site.Router,
			}
			go func(srv *http.Server, siteName string) {
				log.Printf("Serveur TLS autocert démarré pour %s sur port 443", siteName)
				if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
					log.Printf("Erreur TLS autocert %s : %v", siteName, err)
				}
			}(tlsSrv, cfg.ServerName)
		} else if cfg.SSLEnabled {
			if !fileExists(cfg.SSLCertFile) || !fileExists(cfg.SSLKeyFile) {
				log.Printf("Certificat ou clé ssl introuvable pour site %s", cfg.ServerName)
				continue
			}
			httpsSrv := &http.Server{
				Addr:    ":" + cfg.Listen,
				Handler: site.Router,
			}
			go func(srv *http.Server, siteName, certFile, keyFile string) {
				log.Printf("Serveur HTTPS classique démarré pour %s sur port %s", siteName, cfg.Listen)
				if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
					log.Printf("Erreur HTTPS classique %s : %v", siteName, err)
				}
			}(httpsSrv, cfg.ServerName, cfg.SSLCertFile, cfg.SSLKeyFile)
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