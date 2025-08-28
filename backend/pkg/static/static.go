package static

import (
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rakyll/statik/fs"
	
	// 导入生成的静态文件
	_ "kube-node-manager/statik"
)

// StaticFileHandler 创建静态文件处理中间件
func StaticFileHandler() gin.HandlerFunc {
	// 获取嵌入的静态文件系统
	statikFS, err := fs.New()
	if err != nil {
		// 如果静态文件系统初始化失败，返回一个fallback处理器
		return func(c *gin.Context) {
			reqPath := c.Request.URL.Path
			// 如果是API路径，跳过处理
			if strings.HasPrefix(reqPath, "/api/") {
				c.Next()
				return
			}
			// 返回简单的错误页面
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusInternalServerError, `
<!DOCTYPE html>
<html>
<head><title>Service Unavailable</title></head>
<body>
<h1>Static files not available</h1>
<p>The frontend is not properly built or embedded. Please rebuild the application.</p>
<p>API is still available at <a href="/api/v1/health">/api/v1/health</a></p>
</body>
</html>`)
			c.Abort()
		}
	}

	return func(c *gin.Context) {
		// 获取请求路径
		reqPath := c.Request.URL.Path
		
		// 如果是API路径，跳过静态文件处理
		if strings.HasPrefix(reqPath, "/api/") {
			c.Next()
			return
		}
		
		// 处理根路径
		if reqPath == "/" {
			reqPath = "/index.html"
		}
		
		// 尝试打开文件
		file, err := statikFS.Open(reqPath)
		if err != nil {
			// 如果文件不存在，对于SPA应用返回index.html
			if strings.Contains(err.Error(), "not found") {
				file, err = statikFS.Open("/index.html")
				if err != nil {
					c.Status(http.StatusNotFound)
					c.Abort()
					return
				}
			} else {
				c.Status(http.StatusInternalServerError)
				c.Abort()
				return
			}
		}
		defer file.Close()

		// 获取文件信息
		fileInfo, err := file.Stat()
		if err != nil {
			c.Status(http.StatusInternalServerError)
			c.Abort()
			return
		}

		// 设置内容类型
		contentType := getContentType(reqPath)
		if contentType != "" {
			c.Header("Content-Type", contentType)
		}

		// 设置缓存头
		if isStaticAsset(reqPath) {
			c.Header("Cache-Control", "public, max-age=31536000") // 1年
		} else {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		}

		// 服务文件
		http.ServeContent(c.Writer, c.Request, fileInfo.Name(), fileInfo.ModTime(), file)
		c.Abort()
	}
}

// getContentType 根据文件扩展名返回MIME类型
func getContentType(filePath string) string {
	ext := strings.ToLower(path.Ext(filePath))
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return ""
	}
}

// isStaticAsset 判断是否为静态资源
func isStaticAsset(filePath string) bool {
	ext := strings.ToLower(path.Ext(filePath))
	staticExts := []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"}
	for _, staticExt := range staticExts {
		if ext == staticExt {
			return true
		}
	}
	return false
}