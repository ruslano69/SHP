// pkg/converter/context.go
package converter

import (
	"bytes"
	"context"
	"time"

	"golang.org/x/net/html"
)

// ConvertWithContext конвертация с поддержкой контекста
func (c *DefaultConverter) ConvertWithContext(ctx context.Context, input []byte, opts Options) (*Result, error) {
	startTime := time.Now()
	
	result := &Result{
		OriginalSize: int64(len(input)),
	}

	// Проверка отмены
	select {
	case <-ctx.Done():
		if c.metrics != nil {
			c.metrics.RecordError(ErrContextCanceled)
		}
		return nil, NewError(ErrContextCanceled, "context canceled", ctx.Err())
	default:
	}

	// Парсинг HTML
	doc, err := html.Parse(bytes.NewReader(input))
	if err != nil {
		if c.metrics != nil {
			c.metrics.RecordError(ErrParseFailed)
		}
		if opts.StrictMode {
			return nil, NewError(ErrParseFailed, "failed to parse HTML", err)
		}
		result.Errors = append(result.Errors, err)
	}

	// Проверка отмены после парсинга
	select {
	case <-ctx.Done():
		if c.metrics != nil {
			c.metrics.RecordError(ErrContextCanceled)
		}
		return nil, NewError(ErrContextCanceled, "context canceled after parsing", ctx.Err())
	default:
	}

	// Валидация и исправление
	if opts.AutoFix {
		if err := c.fixNodeWithContext(ctx, doc, result, opts); err != nil {
			if c.metrics != nil {
				c.metrics.RecordError(ErrConversionFailed)
			}
			return nil, err
		}
	} else {
		if err := c.validateNodeWithContext(ctx, doc, result); err != nil {
			if c.metrics != nil {
				c.metrics.RecordError(ErrValidationFailed)
			}
			if opts.StrictMode {
				return nil, NewError(ErrValidationFailed, "validation failed", err)
			}
		}
	}

	// Проверка отмены после валидации
	select {
	case <-ctx.Done():
		if c.metrics != nil {
			c.metrics.RecordError(ErrContextCanceled)
		}
		return nil, NewError(ErrContextCanceled, "context canceled after validation", ctx.Err())
	default:
	}

	// Сериализация в XHTML
	var buf bytes.Buffer
	if err := c.renderXHTML(doc, &buf, opts); err != nil {
		if c.metrics != nil {
			c.metrics.RecordError(ErrConversionFailed)
		}
		return nil, NewError(ErrConversionFailed, "failed to render XHTML", err)
	}

	result.Output = buf.Bytes()
	result.FinalSize = int64(len(result.Output))
	result.Success = len(result.Errors) == 0

	// Записываем метрики
	if c.metrics != nil {
		c.metrics.RecordConversion(time.Since(startTime), result.OriginalSize, result.FinalSize)
		for _, change := range result.Changes {
			c.metrics.RecordChange(change.Type)
		}
	}
	
	return result, nil
}

// ValidateWithContext валидация с поддержкой контекста
func (c *DefaultConverter) ValidateWithContext(ctx context.Context, input []byte) error {
	// Проверка отмены
	select {
	case <-ctx.Done():
		return NewError(ErrContextCanceled, "context canceled", ctx.Err())
	default:
	}

	doc, err := html.Parse(bytes.NewReader(input))
	if err != nil {
		return NewError(ErrParseFailed, "failed to parse HTML", err)
	}
	
	result := &Result{}
	return c.validateNodeWithContext(ctx, doc, result)
}

// validateNodeWithContext проверяет узел с учетом контекста
func (c *DefaultConverter) validateNodeWithContext(ctx context.Context, n *html.Node, result *Result) error {
	// Периодическая проверка отмены
	select {
	case <-ctx.Done():
		return NewError(ErrContextCanceled, "context canceled during validation", ctx.Err())
	default:
	}

	// Базовая валидация (переиспользуем существующую логику)
	if err := c.validateNode(n, result); err != nil {
		return err
	}

	return nil
}

// fixNodeWithContext исправляет узел с учетом контекста
func (c *DefaultConverter) fixNodeWithContext(ctx context.Context, n *html.Node, result *Result, opts Options) error {
	// Периодическая проверка отмены
	select {
	case <-ctx.Done():
		return NewError(ErrContextCanceled, "context canceled during fix", ctx.Err())
	default:
	}

	// Базовое исправление (переиспользуем существующую логику)
	c.fixNode(n, result, opts)

	return nil
}
