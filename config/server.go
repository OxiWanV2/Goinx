package config

import (
    "context"
    "fmt"
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

var (
    activeServersMu sync.Mutex
    activeServers   = make(map[string]*SiteServer)
)

type SiteServer struct {
    Config SiteConfig

    httpServer *http.Server
    running    bool
    mu         sync.Mutex
}

func dnsPointsToLocalIP(domain string) bool {
    ips, err := net.LookupIP(domain)
    if err != nil {
        log.Printf("Échec résolution DNS %s : %v", domain, err)
        return false
    }
    for _, ip := range ips {
        if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
            return true
        }
    }
    return false
}

func isSslOnPort80(config SiteConfig) bool {
    return config.SSLEnabled && config.Listen == "80"
}

func StartServer(siteName string, config SiteConfig) error {
    activeServersMu.Lock()
    defer activeServersMu.Unlock()

    if srv, exists := activeServers[siteName]; exists && srv.running {
        return fmt.Errorf("site %s serveur déjà démarré", siteName)
    }

    gin.SetMode(gin.ReleaseMode)
    r := gin.Default()

    r.Static("/", config.Root)

    if config.VuejsRewrite.Path != "" && config.VuejsRewrite.Fallback != "" {
        r.NoRoute(func(c *gin.Context) {
            if strings.HasPrefix(c.Request.URL.Path, config.VuejsRewrite.Path) {
                c.File(filepath.Join(config.Root, config.VuejsRewrite.Fallback))
            } else {
                ServeErrorPage(c, http.StatusNotFound, config)
                c.Abort()
            }
        })
    } else {
        r.NoRoute(func(c *gin.Context) {
            ServeErrorPage(c, http.StatusNotFound, config)
            c.Abort()
        })
    }

    listenPort := config.Listen
    srv := &http.Server{
        Addr:    ":" + listenPort,
        Handler: r,
    }

    // Gestion Let’s Encrypt
    if config.UseLetsEncrypt {
        if !dnsPointsToLocalIP(config.ServerName) {
            log.Printf("Le domaine %s ne pointe pas vers cette IP. Ignorer Let’s Encrypt pour %s.", config.ServerName, siteName)
            // Fallback au serveur classique (HTTP ou SSL classique)
            return startClassicServer(siteName, config, srv, r)
        }

        m := &autocert.Manager{
            Cache:      autocert.DirCache("/etc/goinx/certs-cache"),
            Prompt:     autocert.AcceptTOS,
            HostPolicy: autocert.HostWhitelist(config.ServerName),
        }
        srv.TLSConfig = m.TLSConfig()

        // Serveur HTTP sur port 80 pour challenge Let’s Encrypt
        go func() {
            httpSrv := &http.Server{
                Addr:    ":80",
                Handler: m.HTTPHandler(nil),
            }
            log.Printf("Serveur HTTP Let’s Encrypt démarré sur port 80 pour site %s", siteName)
            if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                log.Printf("Erreur serveur HTTP Let’s Encrypt %s : %v", siteName, err)
            }
        }()

        go func() {
            log.Printf("Serveur HTTPS avec Let’s Encrypt démarré sur port %s pour site %s", config.Listen, siteName)
            if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
                log.Printf("Erreur HTTPS Let’s Encrypt site %s : %v", siteName, err)
                activeServersMu.Lock()
                if srvEntry, ok := activeServers[siteName]; ok {
                    srvEntry.mu.Lock()
                    srvEntry.running = false
                    srvEntry.mu.Unlock()
                }
                activeServersMu.Unlock()
            }
        }()

        activeServers[siteName] = &SiteServer{
            Config:     config,
            httpServer: srv,
            running:    true,
        }

        return nil
    }

    return startClassicServer(siteName, config, srv, r)
}

func startClassicServer(siteName string, config SiteConfig, srv *http.Server, handler http.Handler) error {
    if config.SSLEnabled {
        if _, err := os.Stat(config.SSLCertFile); err != nil {
            return fmt.Errorf("ssl_cert_file introuvable ou inaccessible: %v", err)
        }
        if _, err := os.Stat(config.SSLKeyFile); err != nil {
            return fmt.Errorf("ssl_key_file introuvable ou inaccessible: %v", err)
        }

        if isSslOnPort80(config) {
            httpsPort := "443"
            httpPort := "80"
            srv.Addr = ":" + httpsPort

            startRedirectHttpToHttps(httpPort, httpsPort, siteName)

            go func() {
                log.Printf("Serveur site %s démarré en HTTPS sur port %s\n", siteName, httpsPort)
                if err := srv.ListenAndServeTLS(config.SSLCertFile, config.SSLKeyFile); err != nil && err != http.ErrServerClosed {
                    log.Printf("Erreur HTTPS site %s : %v", siteName, err)
                    activeServersMu.Lock()
                    if srvEntry, ok := activeServers[siteName]; ok {
                        srvEntry.mu.Lock()
                        srvEntry.running = false
                        srvEntry.mu.Unlock()
                    }
                    activeServersMu.Unlock()
                }
            }()
        } else {
            go func() {
                log.Printf("Serveur site %s démarré en HTTPS sur port %s\n", siteName, config.Listen)
                if err := srv.ListenAndServeTLS(config.SSLCertFile, config.SSLKeyFile); err != nil && err != http.ErrServerClosed {
                    log.Printf("Erreur HTTPS site %s : %v", siteName, err)
                    activeServersMu.Lock()
                    if srvEntry, ok := activeServers[siteName]; ok {
                        srvEntry.mu.Lock()
                        srvEntry.running = false
                        srvEntry.mu.Unlock()
                    }
                    activeServersMu.Unlock()
                }
            }()
        }
    } else {
        go func() {
            log.Printf("Serveur site %s démarré sur port %s\n", siteName, config.Listen)
            if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                log.Printf("Erreur HTTP site %s : %v", siteName, err)
                activeServersMu.Lock()
                if srvEntry, ok := activeServers[siteName]; ok {
                    srvEntry.mu.Lock()
                    srvEntry.running = false
                    srvEntry.mu.Unlock()
                }
                activeServersMu.Unlock()
            }
        }()
    }

    activeServers[siteName] = &SiteServer{
        Config:     config,
        httpServer: srv,
        running:    true,
    }

    return nil
}

func startRedirectHttpToHttps(httpPort, httpsPort, siteName string) {
    go func() {
        redirectSrv := &http.Server{
            Addr: ":" + httpPort,
            Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                target := "https://" + r.Host
                if httpsPort != "443" {
                    target += ":" + httpsPort
                }
                target += r.URL.RequestURI()
                http.Redirect(w, r, target, http.StatusMovedPermanently)
            }),
        }
        log.Printf("Redirection HTTP->HTTPS en marche sur port %s pour site %s\n", httpPort, siteName)
        if err := redirectSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("Erreur redirect HTTP->HTTPS pour site %s : %v", siteName, err)
        }
    }()
}

func StopServer(siteName string) error {
    activeServersMu.Lock()
    siteSrv, exists := activeServers[siteName]
    activeServersMu.Unlock()

    if !exists || !siteSrv.running {
        return fmt.Errorf("site %s serveur non démarré", siteName)
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

    log.Printf("Serveur site %s arrêté proprement\n", siteName)
    return nil
}

func IsServerRunning(siteName string) bool {
    activeServersMu.Lock()
    defer activeServersMu.Unlock()
    s, ok := activeServers[siteName]
    return ok && s.running
}