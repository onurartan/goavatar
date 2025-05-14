/*

██████╗  █████╗ ███╗   ██╗ ██████╗
██╔══██╗██╔══██╗████╗  ██║██╔═══██╗
██████╔╝███████║██╔██╗ ██║██║   ██║
██╔══██╗██╔══██║██║╚██╗██║██║   ██║
██████╔╝██║  ██║██║ ╚████║╚██████╔╝
╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝

Author      : BAN0
Project     : GO Avatar - Avatar Generator API
Repository  : github.com/onurartan/goavatar

*/

package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func avatarHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/avatar/")
	if name == "" {
		// http.Error(w, "Missing name parameter", http.StatusBadRequest)
		writeError(w, http.StatusBadRequest, "Missing name parameter = `/avatar/:name`")
		return
	}
	imageResponse(name, w, r)
}

func githubAvatarHandler(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.URL.Path, "/avatar/github/")
	if username == "" {
		// http.Error(w, "Missing GitHub username", http.StatusBadRequest)
		writeError(w, http.StatusBadRequest, "Missing GitHub username = `/avatar/github/:username`")
		return
	}

	name, err := fetchGitHubName(username)
	if err != nil {
		// http.Error(w, fmt.Sprintf("Error fetching GitHub data: %v", err), http.StatusInternalServerError)
		err_message := fmt.Sprintf("Error fetching GitHub data: %v", err)
		writeError(w, http.StatusInternalServerError, err_message)
		return
	}

	imageResponse(name, w, r)
}

func main() {
	printSignature()

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "src/static/index.html")
	})

	mux.HandleFunc("/avatar/", avatarHandler)
	mux.HandleFunc("/avatar/github/", githubAvatarHandler)

	handlerWithCORS := corsMiddleware(mux)

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handlerWithCORS))
}
