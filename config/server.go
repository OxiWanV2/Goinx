package config

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "github.com/gin-gonic/gin"
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

    srv := &http.Server{
        Addr:    ":" + config.Listen,
        Handler: r,
    }

    var err error
    if config.SSLEnabled {
        if _, errCert := os.Stat(config.SSLCertFile); errCert != nil {
            return fmt.Errorf("ssl_cert_file introuvable ou inaccessible: %v", errCert)
        }
        if _, errKey := os.Stat(config.SSLKeyFile); errKey != nil {
            return fmt.Errorf("ssl_key_file introuvable ou inaccessible: %v", errKey)
        }

        go func() {
            log.Printf("Serveur site %s démarré en HTTPS sur port %s\n", siteName, config.Listen)
            if err := srv.ListenAndServeTLS(config.SSLCertFile, config.SSLKeyFile); err != nil && err != http.ErrServerClosed {
                log.Printf("Serveur site %s erreur HTTPS : %v", siteName, err)
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
            log.Printf("Serveur site %s démarré sur port %s\n", siteName, config.Listen)
            if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                log.Printf("Serveur site %s erreur HTTP : %v", siteName, err)
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

    siteSrv := &SiteServer{
        Config:     config,
        httpServer: srv,
        running:    true,
    }
    activeServers[siteName] = siteSrv

    return err
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

    ctx, cancel := context.WithTimeout(context.Background(), 5_000_000_000)
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