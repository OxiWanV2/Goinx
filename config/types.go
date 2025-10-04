package config

type SiteConfig struct {
    ServerName   string       // Nom de domaine ou IP
    Listen       string       // Port d’écoute (exemple "80")
    Root         string       // Chemin vers fichiers statiques
    VuejsRewrite VuejsRewrite // Config rewrite VueJS
	ErrorPagesDir string // Directive pour les pages d'erreur custom
	SSLEnabled   bool // Permet d'activer ou non le SSL
    SSLCertFile  string // Fichier de certificat SSL
    SSLKeyFile   string // Fichier de clef SSL
}

type VuejsRewrite struct {
    Path     string // Exemple : "/"
    Fallback string // Exemple : "index.html"
}