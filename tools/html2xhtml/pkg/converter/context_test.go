// pkg/converter/context_test.go
package converter

import (
	"context"
	"testing"
	"time"
)

func TestConvertWithContext_Success(t *testing.T) {
	conv := New()
	ctx := context.Background()
	input := []byte(`<html><body><p>test</p></body></html>`)

	result, err := conv.ConvertWithContext(ctx, input, Options{AutoFix: true})
	if err != nil {
		t.Fatalf("ConvertWithContext() error = %v", err)
	}

	if !result.Success {
		t.Error("Expected successful conversion")
	}
}

func TestConvertWithContext_Timeout(t *testing.T) {
	conv := New()
	
	// Контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	
	time.Sleep(10 * time.Millisecond) // Убеждаемся что контекст истек
	
	input := []byte(`<html><body><p>test</p></body></html>`)
	
	_, err := conv.ConvertWithContext(ctx, input, Options{AutoFix: true})
	if err == nil {
		t.Error("Expected timeout error")
	}
	
	if convErr, ok := err.(*Error); ok {
		if convErr.Code != ErrContextCanceled {
			t.Errorf("Expected ErrContextCanceled, got %d", convErr.Code)
		}
	}
}

func TestConvertWithContext_Cancel(t *testing.T) {
	conv := New()
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу
	
	input := []byte(`<html><body><p>test</p></body></html>`)
	
	_, err := conv.ConvertWithContext(ctx, input, Options{AutoFix: true})
	if err == nil {
		t.Error("Expected cancellation error")
	}
}

func TestValidateWithContext_Success(t *testing.T) {
	conv := New()
	ctx := context.Background()
	input := []byte(`<html><body><p>test</p><br /></body></html>`)

	err := conv.ValidateWithContext(ctx, input)
	if err != nil {
		t.Errorf("ValidateWithContext() unexpected error = %v", err)
	}
}

func TestValidateWithContext_Cancel(t *testing.T) {
	conv := New()
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	input := []byte(`<html><body><p>test</p></body></html>`)
	
	err := conv.ValidateWithContext(ctx, input)
	if err == nil {
		t.Error("Expected cancellation error")
	}
}

func TestConvertWithContext_Metrics(t *testing.T) {
	metrics := NewMetrics()
	conv := NewWithMetrics(metrics)
	ctx := context.Background()
	
	input := []byte(`<HTML><BODY><BR></BODY></HTML>`)
	
	result, err := conv.ConvertWithContext(ctx, input, Options{AutoFix: true})
	if err != nil {
		t.Fatalf("ConvertWithContext() error = %v", err)
	}
	
	if !result.Success {
		t.Error("Expected successful conversion")
	}
	
	stats := metrics.GetStats()
	if stats.TotalConversions != 1 {
		t.Errorf("Expected 1 conversion, got %d", stats.TotalConversions)
	}
	
	if stats.SuccessfulConversions != 1 {
		t.Errorf("Expected 1 successful conversion, got %d", stats.SuccessfulConversions)
	}
}

func TestConvertWithContext_MetricsError(t *testing.T) {
	metrics := NewMetrics()
	conv := NewWithMetrics(metrics)
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	input := []byte(`<html><body><p>test</p></body></html>`)
	
	_, err := conv.ConvertWithContext(ctx, input, Options{AutoFix: true})
	if err == nil {
		t.Error("Expected error")
	}
	
	stats := metrics.GetStats()
	if stats.FailedConversions != 1 {
		t.Errorf("Expected 1 failed conversion, got %d", stats.FailedConversions)
	}
}

func BenchmarkConvertWithContext(b *testing.B) {
	conv := New()
	ctx := context.Background()
	input := []byte(`<html><body><p>test</p></body></html>`)
	opts := Options{AutoFix: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = conv.ConvertWithContext(ctx, input, opts)
	}
}
