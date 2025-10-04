package backend

import (
    "bufio"
    "fmt"
    "log"
    "os"
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
    backendsMu   sync.Mutex
    backends     = make(map[string]*BackendInstance)
    backendsLogs = make(map[string]chan string)
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

    stdoutPipe, err := cmd.StdoutPipe()
    if err != nil {
        return err
    }
    stderrPipe, err := cmd.StderrPipe()
    if err != nil {
        return err
    }

    logChan := make(chan string, 100)
    backendsMu.Lock()
    backendsLogs[siteName] = logChan
    backendsMu.Unlock()

    go func() {
        scannerOut := bufio.NewScanner(stdoutPipe)
        for scannerOut.Scan() {
            logChan <- scannerOut.Text()
        }
    }()
    go func() {
        scannerErr := bufio.NewScanner(stderrPipe)
        for scannerErr.Scan() {
            logChan <- scannerErr.Text()
        }
    }()

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
        close(logChan)
        delete(backendsLogs, siteName)
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
    close(backendsLogs[siteName])
    delete(backendsLogs, siteName)
    backendsMu.Unlock()
    log.Printf("Backend site %s stoppé", siteName)
    return nil
}

func GetBackendLogChannel(siteName string) (chan string, bool) {
    backendsMu.Lock()
    defer backendsMu.Unlock()
    ch, ok := backendsLogs[siteName]
    return ch, ok
}