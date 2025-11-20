// pkg/converter/prevalidator.go
package converter

import (
	"regexp"
	"strings"
)

// PreValidator проверяет HTML до парсинга для обнаружения нарушений XHTML
type PreValidator struct {
	uppercaseTagRe    *regexp.Regexp
	uppercaseAttrRe   *regexp.Regexp
	unquotedAttrRe    *regexp.Regexp
	unclosedVoidRe    *regexp.Regexp
}

func NewPreValidator() *PreValidator {
	return &PreValidator{
		// Теги в uppercase: <HTML>, <BODY>, <DIV> etc
		uppercaseTagRe: regexp.MustCompile(`</?[A-Z][A-Z0-9]*`),

		// Атрибуты в uppercase: CLASS="test", ID="main"
		uppercaseAttrRe: regexp.MustCompile(`\s+[A-Z][A-Z0-9_-]*=`),

		// Атрибуты без кавычек: src=pic.jpg вместо src="pic.jpg"
		unquotedAttrRe: regexp.MustCompile(`\s+(\w+)=([^"'][^\s>]+)`),

		// Незакрытые void элементы: <br> вместо <br />
		// Проверяем что тег заканчивается на > без / перед ним
		unclosedVoidRe: regexp.MustCompile(`<(br|img|input|meta|link|hr|area|base|col|embed|param|source|track|wbr)(\s[^/>]*|)>`),
	}
}

// ValidationIssue описывает проблему в HTML
type ValidationIssue struct {
	Type     IssueType
	Line     int
	Column   int
	Message  string
	Original string
	Fixed    string
}

type IssueType int

const (
	IssueUppercaseTag IssueType = iota
	IssueUppercaseAttr
	IssueUnquotedAttr
	IssueUnclosedVoid
	IssueInvalidNesting
)

// Validate проверяет HTML и возвращает список проблем
func (pv *PreValidator) Validate(input string) []ValidationIssue {
	var issues []ValidationIssue

	// Проверка uppercase тегов
	if matches := pv.uppercaseTagRe.FindAllString(input, -1); len(matches) > 0 {
		seen := make(map[string]bool)
		for _, match := range matches {
			tagName := strings.TrimPrefix(strings.TrimPrefix(match, "</"), "<")
			if !seen[tagName] {
				seen[tagName] = true
				issues = append(issues, ValidationIssue{
					Type:     IssueUppercaseTag,
					Message:  "Tag must be lowercase",
					Original: tagName,
					Fixed:    strings.ToLower(tagName),
				})
			}
		}
	}

	// Проверка uppercase атрибутов
	if matches := pv.uppercaseAttrRe.FindAllString(input, -1); len(matches) > 0 {
		seen := make(map[string]bool)
		for _, match := range matches {
			attrName := strings.TrimSuffix(strings.TrimSpace(match), "=")
			if !seen[attrName] {
				seen[attrName] = true
				issues = append(issues, ValidationIssue{
					Type:     IssueUppercaseAttr,
					Message:  "Attribute must be lowercase",
					Original: attrName,
					Fixed:    strings.ToLower(attrName),
				})
			}
		}
	}

	// Проверка атрибутов без кавычек
	if matches := pv.unquotedAttrRe.FindAllStringSubmatch(input, -1); len(matches) > 0 {
		seen := make(map[string]bool)
		for _, match := range matches {
			if len(match) >= 3 {
				key := match[0]
				if !seen[key] {
					seen[key] = true
					issues = append(issues, ValidationIssue{
						Type:     IssueUnquotedAttr,
						Message:  "Attribute value must be quoted",
						Original: match[0],
						Fixed:    match[1] + `="` + match[2] + `"`,
					})
				}
			}
		}
	}

	// Проверка незакрытых void элементов
	if matches := pv.unclosedVoidRe.FindAllStringSubmatch(input, -1); len(matches) > 0 {
		seen := make(map[string]bool)
		for _, match := range matches {
			if len(match) >= 1 {
				tagName := match[1]
				if !seen[tagName] {
					seen[tagName] = true
					issues = append(issues, ValidationIssue{
						Type:     IssueUnclosedVoid,
						Message:  "Void element must be self-closing",
						Original: "<" + tagName + ">",
						Fixed:    "<" + tagName + " />",
					})
				}
			}
		}
	}

	return issues
}

// CountIssuesByType подсчитывает количество проблем по типам
func CountIssuesByType(issues []ValidationIssue) map[IssueType]int {
	counts := make(map[IssueType]int)
	for _, issue := range issues {
		counts[issue.Type]++
	}
	return counts
}
