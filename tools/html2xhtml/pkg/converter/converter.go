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
	metrics      Metrics
	preValidator *PreValidator
}

func New() Converter {
	return &DefaultConverter{
		metrics:      &NoOpMetrics{},
		preValidator: NewPreValidator(),
	}
}

func NewWithMetrics(metrics Metrics) Converter {
	return &DefaultConverter{
		metrics:      metrics,
		preValidator: NewPreValidator(),
	}
}

func (c *DefaultConverter) Convert(input []byte, opts Options) (*Result, error) {
	result := &Result{
		OriginalSize: int64(len(input)),
	}

	// ШАГ 1: Пре-валидация для StrictMode (обнаружение проблем ДО нормализации парсером)
	if opts.StrictMode || opts.AutoFix {
		issues := c.preValidator.Validate(string(input))

		if opts.StrictMode && len(issues) > 0 {
			// В строгом режиме первая же проблема - это ошибка
			issue := issues[0]
			return nil, NewError(ErrValidationFailed, issue.Message+": "+issue.Original, nil)
		}

		// Конвертируем issues в Changes для отслеживания
		if opts.AutoFix {
			for _, issue := range issues {
				changeType := ChangeUnclosedTag
				switch issue.Type {
				case IssueUppercaseTag:
					changeType = ChangeUppercaseTag
				case IssueUppercaseAttr, IssueUnquotedAttr:
					changeType = ChangeUnquotedAttr
				case IssueUnclosedVoid:
					changeType = ChangeUnclosedTag
				}

				result.Changes = append(result.Changes, Change{
					Type:     changeType,
					Message:  issue.Message,
					Original: issue.Original,
					Fixed:    issue.Fixed,
				})
			}
		}
	}

	// ШАГ 2: Парсинг HTML (парсер нормализует автоматически)
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

	// ШАГ 3: Валидация структуры (после парсинга)
	if !opts.AutoFix {
		if err := c.validateNode(doc, result); err != nil {
			if c.metrics != nil {
				c.metrics.RecordError(ErrValidationFailed)
			}
			if opts.StrictMode {
				return nil, NewError(ErrValidationFailed, "validation failed", err)
			}
		}
	}

	// ШАГ 4: Сериализация в XHTML
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
		for _, change := range result.Changes {
			c.metrics.RecordChange(change.Type)
		}
	}

	return result, nil
}

func (c *DefaultConverter) Validate(input []byte) error {
	// Пре-валидация: проверка исходного HTML
	issues := c.preValidator.Validate(string(input))
	if len(issues) > 0 {
		issue := issues[0]
		return NewError(ErrValidationFailed, issue.Message+": "+issue.Original, nil)
	}

	// Парсинг и валидация структуры
	doc, err := html.Parse(bytes.NewReader(input))
	if err != nil {
		return NewError(ErrParseFailed, "failed to parse HTML", err)
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
