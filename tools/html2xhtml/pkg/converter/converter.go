// pkg/converter/converter.go
package converter

import (
	"bytes"
	"context"
	"errors"
	"golang.org/x/net/html"
	"io"
	"strings"
)

// Converter интерфейс для конвертации HTML → XHTML
type Converter interface {
	Convert(input []byte, opts Options) (*Result, error)
	Validate(input []byte) error
	ConvertWithContext(ctx context.Context, input []byte, opts Options) (*Result, error)
	ValidateWithContext(ctx context.Context, input []byte) error
}

// Options опции конвертации
type Options struct {
	StrictMode         bool // Строгий режим - отказ при ошибках
	AutoFix            bool // Автоисправление ошибок
	Verbose            bool // Детальные логи
	PreserveFormatting bool // Сохранять форматирование
	ValidateOnly       bool // Только валидация, без конвертации
}

// Result результат конвертации
type Result struct {
	Success      bool
	Output       []byte
	OriginalSize int64
	FinalSize    int64
	Changes      []Change
	Errors       []error
	Warnings     []string
}

// Change описание изменения
type Change struct {
	Type     ChangeType
	Location string // путь в DOM: html>body>div[0]>p
	Message  string
	Original string
	Fixed    string
}

type ChangeType int

const (
	ChangeUnclosedTag ChangeType = iota
	ChangeUnquotedAttr
	ChangeUppercaseTag
	ChangeInvalidNesting
	ChangeMissingNamespace
)

// DefaultConverter реализация конвертера
type DefaultConverter struct{
	metrics Metrics
}

func New() Converter {
	return &DefaultConverter{
		metrics: &NoOpMetrics{},
	}
}

func NewWithMetrics(metrics Metrics) Converter {
	return &DefaultConverter{
		metrics: metrics,
	}
}

func (c *DefaultConverter) Convert(input []byte, opts Options) (*Result, error) {
	result := &Result{
		OriginalSize: int64(len(input)),
	}

	// Парсинг HTML
	doc, err := html.Parse(bytes.NewReader(input))
	if err != nil {
		if opts.StrictMode {
			return nil, err
		}
		result.Errors = append(result.Errors, err)
	}

	// Валидация и исправление
	if opts.AutoFix {
		c.fixNode(doc, result, opts)
	} else if err := c.validateNode(doc, result); err != nil {
		if opts.StrictMode {
			return nil, err
		}
	}

	// Сериализация в XHTML
	var buf bytes.Buffer
	if err := c.renderXHTML(doc, &buf, opts); err != nil {
		return nil, err
	}

	result.Output = buf.Bytes()
	result.FinalSize = int64(len(result.Output))
	result.Success = len(result.Errors) == 0
	
	return result, nil
}

func (c *DefaultConverter) Validate(input []byte) error {
	doc, err := html.Parse(bytes.NewReader(input))
	if err != nil {
		return err
	}
	
	result := &Result{}
	return c.validateNode(doc, result)
}

// validateNode проверяет узел на соответствие XHTML
func (c *DefaultConverter) validateNode(n *html.Node, result *Result) error {
	if n.Type == html.ElementNode {
		// 1. Проверка lowercase тегов
		if n.Data != strings.ToLower(n.Data) {
			return errors.New("tag must be lowercase: " + n.Data)
		}

		// 2. Проверка закрытия void элементов
		if isVoidElement(n.Data) && n.FirstChild != nil {
			return errors.New("void element cannot have children: " + n.Data)
		}

		// 3. Проверка атрибутов
		for _, attr := range n.Attr {
			if attr.Key != strings.ToLower(attr.Key) {
				return errors.New("attribute must be lowercase: " + attr.Key)
			}
		}
	}

	// Рекурсия по детям
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if err := c.validateNode(child, result); err != nil {
			return err
		}
	}

	return nil
}

// fixNode исправляет узел для XHTML
func (c *DefaultConverter) fixNode(n *html.Node, result *Result, opts Options) {
	if n.Type == html.ElementNode {
		// Lowercase тегов
		if n.Data != strings.ToLower(n.Data) {
			result.Changes = append(result.Changes, Change{
				Type:     ChangeUppercaseTag,
				Message:  "Converted tag to lowercase",
				Original: n.Data,
				Fixed:    strings.ToLower(n.Data),
			})
			n.Data = strings.ToLower(n.Data)
		}

		// Lowercase атрибутов
		for i := range n.Attr {
			if n.Attr[i].Key != strings.ToLower(n.Attr[i].Key) {
				n.Attr[i].Key = strings.ToLower(n.Attr[i].Key)
			}
		}
	}

	// Рекурсия
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		c.fixNode(child, result, opts)
	}
}

// renderXHTML сериализует в XHTML формат
func (c *DefaultConverter) renderXHTML(n *html.Node, w io.Writer, opts Options) error {
	switch n.Type {
	case html.DocumentNode:
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if err := c.renderXHTML(child, w, opts); err != nil {
				return err
			}
		}
	case html.ElementNode:
		w.Write([]byte("<" + n.Data))
		
		// Атрибуты
		for _, attr := range n.Attr {
			w.Write([]byte(" " + attr.Key + `="` + html.EscapeString(attr.Val) + `"`))
		}

		// Self-closing для void элементов
		if isVoidElement(n.Data) {
			w.Write([]byte(" />"))
			return nil
		}

		w.Write([]byte(">"))

		// Дети
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if err := c.renderXHTML(child, w, opts); err != nil {
				return err
			}
		}

		w.Write([]byte("</" + n.Data + ">"))
		
	case html.TextNode:
		w.Write([]byte(html.EscapeString(n.Data)))
		
	case html.CommentNode:
		w.Write([]byte("<!--" + n.Data + "-->"))
	}

	return nil
}

// isVoidElement проверяет является ли тег void элементом
func isVoidElement(tag string) bool {
	voidElements := map[string]bool{
		"area": true, "base": true, "br": true, "col": true,
		"embed": true, "hr": true, "img": true, "input": true,
		"link": true, "meta": true, "param": true, "source": true,
		"track": true, "wbr": true,
	}
	return voidElements[tag]
}
