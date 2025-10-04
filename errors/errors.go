package errors

import (
	"log"
	"net/http"
)


func InjectBackendErrorPopup(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		popupHTML := `
<script>
window.onload = function() {
	alert("Attention : backend inaccessible ou erreur critique. Certaines fonctionnalités peuvent être bloquées.");
};
</script>
`

		log.Print("Injection popup d’erreur backend")
		w.Write([]byte(popupHTML))
		next.ServeHTTP(w, r)
	})
}