package config

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/OxiWanV2/Goinx/utils"
)

const (
    sitesAvailableDir = "/etc/goinx/sites-available"
    sitesEnabledDir   = "/etc/goinx/sites-enabled"
)

func EnableSite(siteName string) error {
    src := filepath.Join(sitesAvailableDir, siteName)
    dst := filepath.Join(sitesEnabledDir, siteName)

    if util.Exists(dst) {
        return fmt.Errorf("site %s déjà activé", siteName)
    }
    if !util.Exists(src) {
        return fmt.Errorf("site %s introuvable dans sites-available", siteName)
    }
    return os.Symlink(src, dst)
}

func DisableSite(siteName string) error {
    link := filepath.Join(sitesEnabledDir, siteName)
    if !util.Exists(link) {
        return fmt.Errorf("site %s non activé", siteName)
    }
    return os.Remove(link)
}

func IsSiteEnabled(siteName string) (bool, error) {
    link := filepath.Join(sitesEnabledDir, siteName)
    return util.Exists(link), nil
}