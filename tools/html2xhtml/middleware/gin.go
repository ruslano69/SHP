// middleware/gin.go
package middleware

import (
	"bytes"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ruslano69/shp/pkg/converter"
)

// GinMiddleware для Gin framework
func GinMiddleware(config Config) gin.HandlerFunc {
	if config.Converter == nil {
		if config.EnableMetrics {
			config.metrics = converter.NewMetrics()
			config.Converter = converter.NewWithMetrics(config.metrics)
		} else {
			config.Converter = converter.New()
		}
	}
	if len(config.OnlyExtensions) == 0 {
		config.OnlyExtensions = []string{".html", ".htm"}
	}

	var cache *Cache
	if config.EnableCache {
		cache = newCache()
	}

	return func(c *gin.Context) {
		// Проверка: нужно ли обрабатывать
		if !shouldProcess(c.Request.URL.Path, config) {
			c.Next()
			return
		}

		// Проверка кеша
		if cache != nil {
			if cached, ok := cache.Get(c.Request.URL.Path); ok {
				c.Data(200, "application/xhtml+xml; charset=utf-8", cached)
				c.Abort()
				return
			}
		}

		// Перехват response
		writer := &ginResponseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = writer

		c.Next()

		// Проверка content-type
		contentType := c.Writer.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			// Не HTML, отдаем как есть
			writer.ResponseWriter.Write(writer.body.Bytes())
			return
		}

		// Конвертация с контекстом
		result, err := config.Converter.ConvertWithContext(c.Request.Context(), writer.body.Bytes(), config.Options)
		if err != nil || !result.Success {
			// Ошибка, отдаем оригинал
			writer.ResponseWriter.Write(writer.body.Bytes())
			return
		}

		// Кеширование
		if cache != nil {
			cache.Set(c.Request.URL.Path, result.Output)
		}

		// Отправка XHTML
		c.Writer = writer.ResponseWriter
		c.Data(c.Writer.Status(), "application/xhtml+xml; charset=utf-8", result.Output)
	}
}

type ginResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *ginResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}
