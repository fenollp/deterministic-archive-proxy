package main

import (
	"log"
	"net/http"
	"os"

	dap "github.com/fenollp/deterministic-archive-proxy"
)

func main() {
	iface := ":" + os.Getenv("PORT")
	if iface == ":" {
		iface = ":8080"
	}

	mux := http.NewServeMux()
	mux.Handle("/github.com/", dap.NewGitHubHandler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
	})

	log.Println("Proxy listening on", iface)
	log.Fatalln(http.ListenAndServe(iface, mux))
}
