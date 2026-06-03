package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var pathMap = map[string]string{
	"/":         "/index.html",
	"/login":    "/login.html",
	"/register": "/register.html",
	"/welcome":  "/welcome.html",
	"/video":    "/video.html",
	"/picture":  "/picture.html",
}

// Static 创建一个处理器，用于提供静态文件服务
// root 是静态文件根目录（如 ./resources）
func Static(root string) http.HandlerFunc {
	fs := http.Dir(root)

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// 如果路径在map中，替换为html路径
		if mapped, ok := pathMap[path]; ok {
			r.URL.Path = mapped
		}

		// 尝试打开静态资源
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

		// 查看是否是目录
		stat, err := f.Stat()
		if err != nil {
			http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
			return
		}
		// 禁止访问目录
		if stat.IsDir() {
			http.Error(w, "403 Forbidden", http.StatusForbidden)
			return
		}

		// 根据文件后缀设置 Content-Type（告诉浏览器怎么解析文件）
		// Ext(./login.html) 拿到文件后缀名 .html
		ext := filepath.Ext(r.URL.Path)
		if ct := mimeType(ext); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		// 发送静态文件给浏览器
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), f)
	}
}

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
