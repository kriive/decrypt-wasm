package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/vearutop/statigz"
	"github.com/vearutop/statigz/brotli"
)

//go:embed assets
var st embed.FS

func main() {
	// Auto-index
	withIndexHTML := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/") || len(r.URL.Path) == 0 {
				newpath := path.Join(r.URL.Path, "index.html")
				r.URL.Path = newpath
			}
			h.ServeHTTP(w, r)
		})
	}

	// Retrieve sub directory.
	sub, err := fs.Sub(st, "assets")
	if err != nil {
		log.Fatal(err)
	}

	if err = http.ListenAndServe("0.0.0.0:9090", withIndexHTML(statigz.FileServer(sub.(fs.ReadDirFS), brotli.AddEncoding))); err != nil {
		fmt.Println("Failed to start server", err)
		return
	}
}
