// middleware/http.go
package middleware

import (
	"bytes"
	"net/http"
	"strings"
	"sync"

	"github.com/ruslano69/shp/pkg/converter"
)

// Config конфигурация middleware
type Config struct {
	Converter      converter.Converter
	Options        converter.Options
	EnableCache    bool
	EnableMetrics  bool
	SkipPaths      []string // пути которые пропускаем
	OnlyExtensions []string // только .html по умолчанию
	metrics        converter.Metrics
}

// Cache простой кеш результатов
type Cache struct {
	mu    sync.RWMutex
	items map[string][]byte
}

func newCache() *Cache {
	return &Cache{
		items: make(map[string][]byte),
	}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.items[key]
	return val, ok
}

func (c *Cache) Set(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = val
}

// responseWriter обертка для перехвата response
type responseWriter struct {
	http.ResponseWriter
	buf        *bytes.Buffer
	statusCode int
	written    bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		buf:            &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.buf.Write(b)
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
}

// XHTMLMiddleware для net/http
func XHTMLMiddleware(config Config) func(http.Handler) http.Handler {
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

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Проверка: нужно ли обрабатывать
			if !shouldProcess(r.URL.Path, config) {
				next.ServeHTTP(w, r)
				return
			}

			// Проверка кеша
			if cache != nil {
				if cached, ok := cache.Get(r.URL.Path); ok {
					writeXHTML(w, cached, http.StatusOK)
					return
				}
			}

			// Перехват response
			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r)

			// Проверка content-type
			contentType := rw.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				// Не HTML, отдаем как есть
				copyHeaders(w, rw)
				w.WriteHeader(rw.statusCode)
				w.Write(rw.buf.Bytes())
				return
			}

			// Конвертация с контекстом
			result, err := config.Converter.ConvertWithContext(r.Context(), rw.buf.Bytes(), config.Options)
			if err != nil || !result.Success {
				// Ошибка конвертации, отдаем оригинал
				copyHeaders(w, rw)
				w.WriteHeader(rw.statusCode)
				w.Write(rw.buf.Bytes())
				return
			}

			// Кеширование
			if cache != nil {
				cache.Set(r.URL.Path, result.Output)
			}

			// Отправка XHTML
			writeXHTML(w, result.Output, rw.statusCode)
		})
	}
}

func shouldProcess(path string, config Config) bool {
	// Пропуск путей
	for _, skip := range config.SkipPaths {
		if strings.HasPrefix(path, skip) {
			return false
		}
	}

	// Проверка расширения
	for _, ext := range config.OnlyExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	return false
}

func copyHeaders(dst, src http.ResponseWriter) {
	for k, vv := range src.Header() {
		for _, v := range vv {
			dst.Header().Add(k, v)
		}
	}
}

func writeXHTML(w http.ResponseWriter, body []byte, status int) {
	w.Header().Set("Content-Type", "application/xhtml+xml; charset=utf-8")
	w.Header().Set("X-Converted-By", "SHP-Middleware")
	w.WriteHeader(status)
	w.Write(body)
}

// GetMetrics возвращает метрики из конфигурации
func (c *Config) GetMetrics() converter.ConversionStats {
	if c.metrics != nil {
		return c.metrics.GetStats()
	}
	return converter.ConversionStats{}
}
