package config

type SiteConfig struct {
    ServerName   string       // Nom de domaine ou IP
    Listen       string       // Port d’écoute (exemple "80")
    Root         string       // Chemin vers fichiers statiques
    VuejsRewrite VuejsRewrite // Config rewrite VueJS
}

type VuejsRewrite struct {
    Path     string // Exemple : "/"
    Fallback string // Exemple : "index.html"
}