package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist
var frontendDist embed.FS

func spaHandler() http.Handler {
	subFS, _ := fs.Sub(frontendDist, "dist")
	fileSystem := http.FS(subFS)
	fileServer := http.FileServer(fileSystem)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		f, err := fileSystem.Open(r.URL.Path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
