package backend

import (
	"log"
	"sync"
	"time"
)

type BackendManager struct {
	siteName string
	mu       sync.Mutex
}

func (bm *BackendManager) MonitorBackend() {
	for {
		time.Sleep(30 * time.Second)
		log.Printf("Monitoring backend %s...", bm.siteName)
	}
}