// middleware/echo.go
package middleware

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/ruslano69/shp/pkg/converter"
)

// EchoMiddleware для Echo framework
func EchoMiddleware(config Config) echo.MiddlewareFunc {
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

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Проверка: нужно ли обрабатывать
			if !shouldProcess(c.Request().URL.Path, config) {
				return next(c)
			}

			// Проверка кеша
			if cache != nil {
				if cached, ok := cache.Get(c.Request().URL.Path); ok {
					return c.Blob(200, "application/xhtml+xml; charset=utf-8", cached)
				}
			}

			// Перехват response
			resWriter := &echoResponseWriter{
				ResponseWriter: c.Response().Writer,
				body:           &bytes.Buffer{},
			}
			c.Response().Writer = resWriter

			if err := next(c); err != nil {
				return err
			}

			// Проверка content-type
			contentType := c.Response().Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				// Не HTML, отдаем как есть
				_, err := resWriter.ResponseWriter.Write(resWriter.body.Bytes())
				return err
			}

			// Конвертация с контекстом
			result, err := config.Converter.ConvertWithContext(c.Request().Context(), resWriter.body.Bytes(), config.Options)
			if err != nil || !result.Success {
				// Ошибка, отдаем оригинал
				_, err := resWriter.ResponseWriter.Write(resWriter.body.Bytes())
				return err
			}

			// Кеширование
			if cache != nil {
				cache.Set(c.Request().URL.Path, result.Output)
			}

			// Отправка XHTML
			c.Response().Writer = resWriter.ResponseWriter
			return c.Blob(c.Response().Status, "application/xhtml+xml; charset=utf-8", result.Output)
		}
	}
}

type echoResponseWriter struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (w *echoResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}
