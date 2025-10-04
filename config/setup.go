package config

import (
    "fmt"
    "io"
    "os"
    "os/user"
    "path/filepath"
    "log"
)

func SetupGoinx() error {
    dirs := []string{
        "/etc/goinx",
        "/etc/goinx/sites-available",
        "/etc/goinx/sites-enabled",
    }

    for _, dir := range dirs {
        if _, err := os.Stat(dir); os.IsNotExist(err) {
            err = os.MkdirAll(dir, 0755)
            if err != nil {
                return fmt.Errorf("échec création dossier %s : %v", dir, err)
            }
            log.Printf("Création dossier %s", dir)
        }
    }

    exempleDest := "/etc/goinx/sites-available/exemple"
    if _, err := os.Stat(exempleDest); os.IsNotExist(err) {
        err = CopyDir("./exemple", exempleDest)
        if err != nil {
            return fmt.Errorf("échec copie dossier exemple: %v", err)
        }
        log.Printf("Copie dossier exemple terminée dans %s", exempleDest)
    } else {
        log.Printf("Dossier exemple existe déjà, copie ignorée")
    }

    symlink := "/etc/goinx/sites-enabled/exemple"
    if _, err := os.Lstat(symlink); os.IsNotExist(err) {
        err = os.Symlink(exempleDest, symlink)
        if err != nil {
            return fmt.Errorf("échec création lien symbolique %s: %v", symlink, err)
        }
        log.Printf("Lien symbolique créé: %s -> %s", symlink, exempleDest)
    } else {
        log.Printf("Lien symbolique déjà présent: %s", symlink)
    }

    groupe := "goinx"
    err := createGroup(groupe)
    if err != nil {
        log.Printf("Warning groupe '%s' : %v", groupe, err)
    }

    return nil
}

func CopyDir(src, dest string) error {
    return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        rel, err := filepath.Rel(src, path)
        if err != nil {
            return err
        }
        target := filepath.Join(dest, rel)
        if info.IsDir() {
            return os.MkdirAll(target, info.Mode())
        }
        return copyFile(path, target)
    })
}

func copyFile(srcFile, destFile string) error {
    src, err := os.Open(srcFile)
    if err != nil {
        return err
    }
    defer src.Close()

    dest, err := os.Create(destFile)
    if err != nil {
        return err
    }
    defer dest.Close()

    _, err = io.Copy(dest, src)
    if err != nil {
        return err
    }

    info, err := os.Stat(srcFile)
    if err != nil {
        return err
    }
    return os.Chmod(destFile, info.Mode())
}

func createGroup(name string) error {
    _, err := user.LookupGroup(name)
    if err == nil {
        return nil
    }
    err = execCommand("groupadd", name)
    if err != nil {
        return fmt.Errorf("impossible de créer groupe %s : %v", name, err)
    }
    return nil
}

func execCommand(name string, arg ...string) error {
    cmd := exec.Command(name, arg...)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("commande %s échouée : %s", name, string(out))
    }
    return nil
}