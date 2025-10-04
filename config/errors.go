package config

import (
    "fmt"
    "net/http"
    "os"
    "path/filepath"

    "github.com/gin-gonic/gin"
)

func ServeErrorPage(c *gin.Context, code int, siteConfig SiteConfig) {
    errorsDir := filepath.Join(siteConfig.Root, "errors")
    errPage := filepath.Join(errorsDir, fmt.Sprintf("%d.html", code))

    info, err := os.Stat(errPage)
    if err == nil && !info.IsDir() {
        c.Status(code)
        c.File(errPage)
        return
    }

    c.Data(code, "text/html; charset=utf-8", []byte(defaultErrorPage(code)))
}

func defaultErrorPage(code int) string {
    message := http.StatusText(code)
    if message == "" {
        message = "Erreur inconnue"
    }
    return fmt.Sprintf(`<!DOCTYPE html>
<html lang="fr">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Goinx | %d %s</title>
    <script src="https://unpkg.com/vue@3/dist/vue.global.prod.js"></script>
    <style>
        :root {
            --background: #ffffff;
            --foreground: #000000;
            --card: #f8f9fa;
            --card-foreground: #1f2937;
            --primary: #000000;
            --secondary: #f1f5f9;
            --muted: #64748b;
            --border: #e2e8f0;
            --shadow: rgba(0, 0, 0, 0.1);
        }

        @media (prefers-color-scheme: dark) {
            :root {
                --background: #000000;
                --foreground: #ffffff;
                --card: #0a0a0a;
                --card-foreground: #f8fafc;
                --primary: #ffffff;
                --secondary: #1e293b;
                --muted: #94a3b8;
                --border: #27272a;
                --shadow: rgba(255, 255, 255, 0.05);
            }
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            background-color: var(--background);
            color: var(--foreground);
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 1rem;
            transition: background-color 0.3s ease, color 0.3s ease;
        }

        .error-container {
            background: var(--card);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 3rem 2rem;
            text-align: center;
            max-width: 480px;
            width: 100%%;
            box-shadow: 0 4px 6px -1px var(--shadow), 0 2px 4px -1px var(--shadow);
            transition: all 0.3s ease;
        }

        .error-code {
            font-size: 6rem;
            font-weight: 800;
            color: var(--primary);
            margin-bottom: 1rem;
            line-height: 1;
            letter-spacing: -0.025em;
        }

        .error-message {
            font-size: 1.5rem;
            color: var(--card-foreground);
            margin-bottom: 1.5rem;
            font-weight: 500;
        }

        .error-description {
            color: var(--muted);
            margin-bottom: 2rem;
            font-size: 0.95rem;
        }

        .actions {
            display: flex;
            gap: 1rem;
            justify-content: center;
            flex-wrap: wrap;
        }

        .btn {
            padding: 0.75rem 1.5rem;
            border-radius: 8px;
            font-size: 0.875rem;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s ease;
            text-decoration: none;
            display: inline-flex;
            align-items: center;
            gap: 0.5rem;
            border: none;
        }

		a {
			text-decoration: underline;
			color: var(--muted);
		}

        .btn-primary {
            background: var(--primary);
            color: var(--background);
        }

        .btn-primary:hover {
            opacity: 0.9;
            transform: translateY(-1px);
        }

        .btn-secondary {
            background: var(--secondary);
            color: var(--card-foreground);
            border: 1px solid var(--border);
        }

        .btn-secondary:hover {
            background: var(--muted);
            transform: translateY(-1px);
        }

        .server-info {
            margin-top: 2rem;
            padding-top: 1.5rem;
            border-top: 1px solid var(--border);
            color: var(--muted);
            font-size: 0.8rem;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 0.5rem;
        }

        .pulse {
            animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
        }

        @keyframes pulse {
            0%%, 100%% {
                opacity: 1;
            }
            50%% {
                opacity: 0.5;
            }
        }

        .fade-in {
            animation: fadeIn 0.6s ease-out;
        }

        @keyframes fadeIn {
            from {
                opacity: 0;
                transform: translateY(20px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        @media (max-width: 640px) {
            .error-code {
                font-size: 4rem;
            }
            .error-message {
                font-size: 1.25rem;
            }
            .error-container {
                padding: 2rem 1.5rem;
            }
            .actions {
                flex-direction: column;
            }
        }
    </style>
</head>
<body>
    <div id="app">
        <div class="error-container fade-in">
            <div class="error-code pulse">{{ errorCode }}</div>
            <h1 class="error-message">{{ errorMessage }}</h1>
            <p class="error-description">{{ description }}</p>
            
            <div class="actions">
                <button @click="goHome" class="btn btn-primary">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
                        <polyline points="9,22 9,12 15,12 15,22"/>
                    </svg>
                    Retourner à l'accueil
                </button>
                <button @click="goBack" class="btn btn-secondary">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                        <polyline points="15,18 9,12 15,6"/>
                    </svg>
                    Retour
                </button>
            </div>

            <div class="server-info">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
                    <line x1="8" y1="21" x2="16" y2="21"/>
                    <line x1="12" y1="17" x2="12" y2="21"/>
                </svg>
                <a href="https://github.com/OxiWanV2/Goinx" >Propulsé par Goinx Web Server</a>
            </div>
        </div>
    </div>

    <script>
        const { createApp } = Vue;
        
        createApp({
            data() {
                return {
                    errorCode: %d,
                    errorMessage: '%s',
                    isDarkMode: false
                }
            },
            computed: {
                description() {
                    const descriptions = {
                        400: 'La requête ne peut pas être traitée en raison d\'une syntaxe incorrecte.',
                        401: 'Vous devez vous authentifier pour accéder à cette ressource.',
                        403: 'Vous n\'avez pas l\'autorisation d\'accéder à cette ressource.',
                        404: 'La page que vous cherchez est introuvable. Elle a peut-être été déplacée ou supprimée.',
                        500: 'Une erreur interne du serveur s\'est produite. Veuillez réessayer plus tard.',
                        502: 'Le serveur a reçu une réponse invalide d\'un serveur en amont.',
                        503: 'Le service est temporairement indisponible. Veuillez réessayer plus tard.'
                    };
                    return descriptions[this.errorCode] || 'Une erreur inattendue s\'est produite.';
                }
            },
            methods: {
                goHome() {
                    window.location.href = '/';
                },
                goBack() {
                    if (window.history.length > 1) {
                        window.history.back();
                    } else {
                        window.location.href = '/';
                    }
                },
                detectTheme() {
                    this.isDarkMode = window.matchMedia('(prefers-color-scheme: dark)').matches;
                },
                watchThemeChanges() {
                    window.matchMedia('(prefers-color-scheme: dark)')
                        .addEventListener('change', (e) => {
                            this.isDarkMode = e.matches;
                        });
                }
            },
            mounted() {
                this.detectTheme();
                this.watchThemeChanges();
            }
        }).mount('#app');
    </script>
</body>
</html>`, code, message, code, message)
}