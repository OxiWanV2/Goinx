package util

import (
    "fmt"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
)

func Exists(path string) bool {
    _, err := os.Stat(path)
    return !os.IsNotExist(err)
}

func IsDir(path string) bool {
    info, err := os.Stat(path)
    if err != nil {
        return false
    }
    return info.IsDir()
}

func CreateDirIfNotExist(path string) error {
    if !Exists(path) {
        return os.MkdirAll(path, 0755)
    }
    return nil
}

func CopyFile(src, dst string) error {
    input, err := os.Open(src)
    if err != nil {
        return err
    }
    defer input.Close()

    output, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer output.Close()

    _, err = output.ReadFrom(input)
    if err != nil {
        return err
    }

    info, err := os.Stat(src)
    if err != nil {
        return err
    }
    return os.Chmod(dst, info.Mode())
}

func CopyDir(src, dst string) error {
    return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        relPath, err := filepath.Rel(src, path)
        if err != nil {
            return err
        }
        targetPath := filepath.Join(dst, relPath)

        if info.IsDir() {
            return os.MkdirAll(targetPath, info.Mode())
        }
        return CopyFile(path, targetPath)
    })
}

func RunCommand(name string, args ...string) (string, error) {
    cmd := exec.Command(name, args...)
    out, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("commande %s %v échouée : %s", name, args, string(out))
    }
    return string(out), nil
}

func RemoveFile(path string) error {
    if Exists(path) {
        return os.Remove(path)
    }
    return nil
}

func LinkExists(path string) bool {
    info, err := os.Lstat(path)
    if err != nil {
        return false
    }
    return info.Mode()&os.ModeSymlink != 0
}

func SleepPause(seconds int) {
    time.Sleep(time.Duration(seconds) * time.Second)
}

func Logger(msg string) {
    log.Println(msg)
}

func CleanString(s string) string {
    s = strings.TrimSpace(s)
    s = strings.Join(strings.Fields(s), " ")
    return s
}

func GetEnv(key, def string) string {
    val := os.Getenv(key)
    if val == "" {
        return def
    }
    return val
}