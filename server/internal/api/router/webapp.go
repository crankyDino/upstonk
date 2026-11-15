package router

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// PublicConfig is what the browser app needs at runtime. Only public values:
// the Supabase project URL and its publishable (anon) key.
type PublicConfig struct {
	SupabaseURL            string `json:"supabaseUrl"`
	SupabasePublishableKey string `json:"supabasePublishableKey"`
}

// WithWebApp serves the built GUI (index.html plus assets) from web, with a
// fallback to index.html for client-side routes, and /config.js with cfg.
// This is the single-instance deployment: one server for GUI and API.
func WithWebApp(web fs.FS, cfg PublicConfig) Option {
	return func(o *options) {
		o.web = web
		o.publicConfig = cfg
	}
}

func configJS(cfg PublicConfig) http.HandlerFunc {
	body, _ := json.Marshal(cfg)
	script := "window.__UPSTONK_CONFIG__ = " + string(body) + ";\n"
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write([]byte(script))
	}
}

// spa serves a file when it exists, otherwise index.html (client routes).
func spa(web fs.FS) http.Handler {
	files := http.FileServer(http.FS(web))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name != "" {
			if info, err := fs.Stat(web, name); err == nil && !info.IsDir() {
				if strings.HasPrefix(name, "assets/") { // content-hashed by Vite
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
		}
		index, err := fs.ReadFile(web, "index.html")
		if err != nil {
			http.Error(w, "GUI not built", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	})
}

func apiNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "NOT_FOUND", "code": "NOT_FOUND", "message": "No such API endpoint"})
}
