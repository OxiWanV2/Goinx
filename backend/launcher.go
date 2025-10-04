package backend

import (
	"log"
	"os"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
)

type BackendInstance struct {
	SiteName string
	Cmd      *exec.Cmd
	mu       sync.Mutex
	Running  bool
}

var (
	backendsMu sync.Mutex
	backends   = make(map[string]*BackendInstance)
)

func LaunchNodeBackend(siteName, backendDir, backendFile string) error {
    if backendFile == "" {
        log.Printf("BackendFile vide pour site %s, impossible de lancer le backend", siteName)
        return fmt.Errorf("backendFile vide pour site %s", siteName)
    }

    fullPath := filepath.Join(backendDir, backendFile)

    backendsMu.Lock()
    if bi, exists := backends[siteName]; exists && bi.Running {
        backendsMu.Unlock()
        log.Printf("Backend déjà en cours pour site %s", siteName)
        return nil
    }
    backendsMu.Unlock()

    cmd := exec.Command("node", fullPath)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Start(); err != nil {
        return err
    }

    bi := &BackendInstance{
        SiteName: siteName,
        Cmd:      cmd,
        Running:  true,
    }

    backendsMu.Lock()
    backends[siteName] = bi
    backendsMu.Unlock()

    go func() {
        err := cmd.Wait()
        backendsMu.Lock()
        bi.Running = false
        backendsMu.Unlock()

        if err != nil {
            log.Printf("Backend nodejs site %s fermé avec erreur : %v", siteName, err)
        } else {
            log.Printf("Backend nodejs site %s arrêté proprement", siteName)
        }
    }()

    log.Printf("Backend nodejs démarré pour site %s (%s)", siteName, fullPath)
    return nil
}

func StopBackend(siteName string) error {
	backendsMu.Lock()
	bi, exists := backends[siteName]
	backendsMu.Unlock()
	if !exists || !bi.Running {
		return nil
	}
	if err := bi.Cmd.Process.Kill(); err != nil {
		return err
	}
	backendsMu.Lock()
	bi.Running = false
	backendsMu.Unlock()
	log.Printf("Backend site %s stoppé", siteName)
	return nil
}