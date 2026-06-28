package helpers

import (
	"net/http"
	"os"
)

func CheckOrigin(r *http.Request) bool {
	allowed := os.Getenv("CORS_ORIGIN")
	if allowed == "" || allowed == "*" {
		return true
	}
	return r.Header.Get("Origin") == allowed
}
