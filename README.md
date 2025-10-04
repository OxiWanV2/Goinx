# Goinx

Goinx est un gestionnaire multi-sites orienter VueJS léger et flexible, développé en Go, avec prise en charge complète des sites HTTP et HTTPS, intégration automatique de certificats SSL Let’s Encrypt, et gestion facile via CLI.

***

## Ce que ça fait

- Gère plusieurs sites dynamiques via activation/désactivation par simples liens symboliques (dossiers `sites-enabled`).
- Chaque site a sa propre config : port, racine, règles spécifiques (comme le fallback VueJS), dossiers d’erreurs personnalisées.
- Serveur HTTP frontal unique sur le port 80 qui redirige vers HTTPS et gère les challenges Let’s Encrypt automatiquement.
- Serveur HTTPS unique qui sert tous les sites configurés avec Let’s Encrypt, génération automatique et renouvellement des certificats.
- Support SSL manuel : si tu as déjà tes certificats, tu peux configurer tes sites avec, même sans Let’s Encrypt.
- CLI simple avec les commandes : `list`, `enable`, `disable`, `testconf`, `reload`, `help`, `exit`.
- Reload dynamique de la config et des serveurs sans downtime.
- Pages d’erreur modernes stylisées, fallback VueJS réactif, support multi-code erreurs.
- Architecture modulaire et propre, facile à étendre.

***

## Pourquoi Goinx ?

Tu veux gérer plusieurs sites, en dev ou prod, avec un outil léger et vraiment simple à configurer.  
Tu veux SSL, Let’s Encrypt, ou tes certificats persos, le tout automatisé.  
Tu veux pas te prendre la tête avec Nginx ou Apache.  
Tu veux un CLI efficace et une expérience développeur agréable.  

Goinx est fait pour toi.

***

## Installation rapide

1. Clone ce repo  
2. Place tes configs dans `/etc/goinx/sites-available/`, crée des liens dans `/etc/goinx/sites-enabled/` pour activer  
3. Lance le serveur en mode CLI pour tester :  
```bash
go run cmd/goinx/main.go -cli
```
4. Utilise les commandes `enable`, `disable`, `reload`, etc.  
5. Mets `UseLetsEncrypt=true` dans la config pour générer les certificats automatiquement, ou renseigne tes fichiers de certifs manuels.  
6. En prod, lance le binaire directement sans le flag cli.

***

## Configuration d’un site

Le fichier `.conf` ressemble à ça par exemple :

```ini
# exemple.conf - configuration simple pour le site d'exemple

server_name localhost
listen 80

root /etc/goinx/sites-available/exemple/frontend

# Directive spécifique de Goinx pour les SPA en VueJS
vuejs_rewrite / index.html

# -- Pages erreur (CUSTOM) --
#
# error_pages_dir /etc/goinx/sites-available/exemple/errors
#
# Il va automatiquement en fonction du code erreur charger (ex: [404] -> 404.html) depuis le dossier


# -- Certificat SSL (CUSTOM) --
# ssl_enabled true
# ssl_cert_file /etc/ssl/certs/example.crt
# ssl_key_file /etc/ssl/private/example.key
#
# Important mettez bien le listen en 443 pour les certificats manuelle !
#
# -- Certificat SSL (Letsencrypt) --
#
# use_lets_encrypt true
#
# Ne pas mettre : ["ssl_cert_file" est "ssl_key_file"] Si utilisation de letsencrypt
```

Tu peux :

- Activer Let’s Encrypt avec `UseLetsEncrypt=true`.
- Utiliser un certificat SSL classique avec `SSLEnabled=true` et renseigner `SSLCertFile` / `SSLKeyFile`.
- Faire du fallback VueJS pour une SPA.
- Personnaliser les pages d’erreur.

***

## Fonctionnalités CLI

- `list` : affiche les sites disponibles et leur état.  
- `enable <site>` : active un site (crée un lien dans sites-enabled, initialise).  
- `disable <site>` : désactive un site (supprime le lien, arrête serveur).  
- `reload` : recharge et redémarre les serveurs HTTP/HTTPS sans downtime.  
- `testconf <site>` : teste la config d’un site.  
- `exit` : quitte le CLI.

***

## Architecture technique

- Serveur HTTP frontal unique écoute sur `:80`, gère redirections vers HTTPS et challenges ACME.
- Serveur HTTPS frontal unique écoute sur `:443`, utilise `golang.org/x/crypto/acme/autocert` pour Let’s Encrypt.
- Map `sites` stocke la config et les routers Gin.
- Cache local pour certificats in `/etc/goinx/certs-cache`.
- Gestion fine de la concurrence avec mutex pour éviter les conflits.
- Reload propre géré par arrêt et redémarrage contrôlé des serveurs.
- Support multi-SSL : auto via Let’s Encrypt + manuel avec certificats déjà prêts.

***

## À venir

- Backend applicatif/proxy avancé.  
- Interface web administrateur simple.  
- Monitoring & logs avancés.  
- Tests unitaires & pipeline CI/CD.  
- Optimisations et hardening.

***

## Contribuer

Fork, patch, soumets une PR.  
On discute, on améliore ensemble.

***

## Licence

MIT

***

T’as une question, un bug, un besoin, dis-moi.  
Goinx, c’est fait pour faciliter la vie !
