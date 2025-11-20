// pkg/converter/converter_test.go
package converter

import (
	"strings"
	"testing"
)

func TestConvert_AutoFix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		changes  int
	}{
		{
			name:     "unclosed br tag",
			input:    `<html><body><br></body></html>`,
			expected: `<html><head></head><body><br /></body></html>`,
			changes:  1,
		},
		{
			name:     "uppercase tags",
			input:    `<HTML><BODY><DIV>test</DIV></BODY></HTML>`,
			expected: `<html><head></head><body><div>test</div></body></html>`,
			changes:  3,
		},
		{
			name:     "unquoted attributes",
			input:    `<img src=pic.jpg width=100>`,
			expected: `<html><head></head><body><img src="pic.jpg" width="100" /></body></html>`,
			changes:  2,
		},
		{
			name:     "mixed case attributes",
			input:    `<div CLASS="test" ID="main">content</div>`,
			expected: `<html><head></head><body><div class="test" id="main">content</div></body></html>`,
			changes:  2,
		},
		{
			name:     "multiple void elements",
			input:    `<html><head><meta charset=utf-8><link rel=stylesheet></head></html>`,
			expected: `<html><head><meta charset="utf-8" /><link rel="stylesheet" /></head><body></body></html>`,
			changes:  2,
		},
		{
			name:     "nested unclosed tags",
			input:    `<div><p>text<br><span>more</span></div>`,
			expected: `<html><head></head><body><div><p>text<br /><span>more</span></p></div></body></html>`,
			changes:  1,
		},
		{
			name:     "special characters in text",
			input:    `<p>A & B < C > D</p>`,
			expected: `<html><head></head><body><p>A &amp; B &lt; C &gt; D</p></body></html>`,
			changes:  0,
		},
	}

	conv := New()
	opts := Options{
		StrictMode: false,
		AutoFix:    true,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := conv.Convert([]byte(tt.input), opts)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}

			if !result.Success {
				t.Errorf("Convert() failed, errors: %v", result.Errors)
			}

			got := strings.TrimSpace(string(result.Output))
			want := strings.TrimSpace(tt.expected)

			if got != want {
				t.Errorf("Convert() output mismatch\ngot:  %s\nwant: %s", got, want)
			}

			if len(result.Changes) < tt.changes {
				t.Errorf("Expected at least %d changes, got %d", tt.changes, len(result.Changes))
			}
		})
	}
}

func TestValidate_Strict(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid XHTML",
			input:   `<html><body><p>test</p><br /></body></html>`,
			wantErr: false,
		},
		{
			name:    "uppercase tag",
			input:   `<HTML><body>test</body></HTML>`,
			wantErr: true,
			errMsg:  "Tag must be lowercase",
		},
		{
			name:    "unclosed void element",
			input:   `<html><body><br></body></html>`,
			wantErr: true,
		},
		{
			name:    "uppercase attribute",
			input:   `<div CLASS="test">content</div>`,
			wantErr: true,
			errMsg:  "Attribute must be lowercase",
		},
		{
			name:    "void element with children",
			input:   `<br>text</br>`,
			wantErr: true,
			errMsg:  "Void element must be self-closing",
		},
	}

	conv := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := conv.Validate([]byte(tt.input))
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error message = %v, want substring %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestConvert_StrictMode(t *testing.T) {
	conv := New()
	opts := Options{
		StrictMode: true,
		AutoFix:    false,
	}

	invalidHTML := `<HTML><BODY><BR></BODY></HTML>`
	
	_, err := conv.Convert([]byte(invalidHTML), opts)
	if err == nil {
		t.Error("Expected error in strict mode with invalid HTML")
	}
}

func TestVoidElements(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"br", true},
		{"img", true},
		{"input", true},
		{"meta", true},
		{"link", true},
		{"div", false},
		{"span", false},
		{"p", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := isVoidElement(tt.tag)
			if got != tt.want {
				t.Errorf("isVoidElement(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestResult_Statistics(t *testing.T) {
	conv := New()
	input := `<HTML><BODY><BR><IMG src=test.jpg></BODY></HTML>`
	
	result, err := conv.Convert([]byte(input), Options{AutoFix: true})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if result.OriginalSize != int64(len(input)) {
		t.Errorf("OriginalSize = %d, want %d", result.OriginalSize, len(input))
	}

	if result.FinalSize == 0 {
		t.Error("FinalSize should not be zero")
	}

	if len(result.Changes) == 0 {
		t.Error("Expected some changes to be recorded")
	}

	t.Logf("Changes detected: %d", len(result.Changes))
	for _, change := range result.Changes {
		t.Logf("  - %s: %s → %s", change.Message, change.Original, change.Fixed)
	}
}

func BenchmarkConvert_Small(b *testing.B) {
	conv := New()
	input := []byte(`<html><body><p>test</p></body></html>`)
	opts := Options{AutoFix: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = conv.Convert(input, opts)
	}
}

func BenchmarkConvert_Large(b *testing.B) {
	conv := New()
	// Имитация большого документа
	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := 0; i < 1000; i++ {
		sb.WriteString("<div><p>text</p><br><span>more</span></div>")
	}
	sb.WriteString("</body></html>")
	
	input := []byte(sb.String())
	opts := Options{AutoFix: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = conv.Convert(input, opts)
	}
}
