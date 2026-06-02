package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// pathMap defines short-path → html file mappings, matching C++ DEFAULT_HTML_TAG.
var pathMap = map[string]string{
	"/":         "/index.html",
	"/login":    "/login.html",
	"/register": "/register.html",
	"/welcome":  "/welcome.html",
	"/video":    "/video.html",
	"/picture":  "/picture.html",
}

// Static creates a handler that serves files from root, with short-path expansion.
// Static asset paths (/css/, /js/, /images/, /fonts/) are served directly.
func Static(root string) http.HandlerFunc {
	fs := http.Dir(root)

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check short-path mapping
		if mapped, ok := pathMap[path]; ok {
			r.URL.Path = mapped
		}

		// Try to open the file
		f, err := fs.Open(r.URL.Path)
		if err != nil {
			if os.IsNotExist(err) {
				http.Error(w, "404 Not Found", http.StatusNotFound)
			} else {
				http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			}
			return
		}
		defer f.Close()

		// Check if it's a directory
		stat, err := f.Stat()
		if err != nil {
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}
		if stat.IsDir() {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}

		// Set Content-Type based on extension
		ext := filepath.Ext(r.URL.Path)
		if ct := mimeType(ext); ct != "" {
			w.Header().Set("Content-Type", ct)
		}

		http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
	}
}

// mimeType maps file extensions to Content-Type, matching the C++ SUFFIX_TYPE map.
func mimeType(ext string) string {
	switch strings.ToLower(ext) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json"
	case ".xml":
		return "text/xml"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	case ".pdf":
		return "application/pdf"
	case ".mp4":
		return "video/mp4"
	case ".mpeg":
		return "video/mpeg"
	case ".avi":
		return "video/x-msvideo"
	case ".gz":
		return "application/gzip"
	case ".tar":
		return "application/x-tar"
	case ".woff", ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	default:
		return "application/octet-stream"
	}
}
