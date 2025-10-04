package backend

import (
	"log"
	"os/exec"
)

func SetupNodeModules(backendPath string) error {
	cmd := exec.Command("npm", "install")
	cmd.Dir = backendPath
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	log.Printf("Installation des modules npm dans %s", backendPath)
	return cmd.Run()
}