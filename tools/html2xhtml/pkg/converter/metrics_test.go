// pkg/converter/metrics_test.go
package converter

import (
	"testing"
	"time"
)

func TestMetrics_RecordConversion(t *testing.T) {
	metrics := NewMetrics()
	
	metrics.RecordConversion(100*time.Millisecond, 1000, 900)
	metrics.RecordConversion(200*time.Millisecond, 2000, 1800)
	
	stats := metrics.GetStats()
	
	if stats.TotalConversions != 2 {
		t.Errorf("Expected 2 conversions, got %d", stats.TotalConversions)
	}
	
	if stats.SuccessfulConversions != 2 {
		t.Errorf("Expected 2 successful, got %d", stats.SuccessfulConversions)
	}
	
	if stats.TotalBytesProcessed != 3000 {
		t.Errorf("Expected 3000 bytes processed, got %d", stats.TotalBytesProcessed)
	}
	
	if stats.TotalBytesOutput != 2700 {
		t.Errorf("Expected 2700 bytes output, got %d", stats.TotalBytesOutput)
	}
	
	// Средняя длительность должна быть 150ms
	expectedAvg := 150 * time.Millisecond
	if stats.AverageDuration != expectedAvg {
		t.Errorf("Expected avg duration %v, got %v", expectedAvg, stats.AverageDuration)
	}
}

func TestMetrics_RecordError(t *testing.T) {
	metrics := NewMetrics()
	
	metrics.RecordError(ErrParseFailed)
	metrics.RecordError(ErrValidationFailed)
	metrics.RecordError(ErrParseFailed)
	
	stats := metrics.GetStats()
	
	if stats.TotalConversions != 3 {
		t.Errorf("Expected 3 conversions, got %d", stats.TotalConversions)
	}
	
	if stats.FailedConversions != 3 {
		t.Errorf("Expected 3 failed, got %d", stats.FailedConversions)
	}
	
	if stats.ErrorsByType[ErrParseFailed] != 2 {
		t.Errorf("Expected 2 parse errors, got %d", stats.ErrorsByType[ErrParseFailed])
	}
	
	if stats.ErrorsByType[ErrValidationFailed] != 1 {
		t.Errorf("Expected 1 validation error, got %d", stats.ErrorsByType[ErrValidationFailed])
	}
}

func TestMetrics_RecordChange(t *testing.T) {
	metrics := NewMetrics()
	
	metrics.RecordChange(ChangeUppercaseTag)
	metrics.RecordChange(ChangeUppercaseTag)
	metrics.RecordChange(ChangeUnquotedAttr)
	
	stats := metrics.GetStats()
	
	if stats.ChangesApplied[ChangeUppercaseTag] != 2 {
		t.Errorf("Expected 2 uppercase changes, got %d", stats.ChangesApplied[ChangeUppercaseTag])
	}
	
	if stats.ChangesApplied[ChangeUnquotedAttr] != 1 {
		t.Errorf("Expected 1 unquoted attr change, got %d", stats.ChangesApplied[ChangeUnquotedAttr])
	}
}

func TestMetrics_Reset(t *testing.T) {
	metrics := NewMetrics()
	
	metrics.RecordConversion(100*time.Millisecond, 1000, 900)
	metrics.RecordError(ErrParseFailed)
	metrics.RecordChange(ChangeUppercaseTag)
	
	metrics.Reset()
	
	stats := metrics.GetStats()
	
	if stats.TotalConversions != 0 {
		t.Errorf("Expected 0 conversions after reset, got %d", stats.TotalConversions)
	}
	
	if len(stats.ChangesApplied) != 0 {
		t.Error("Expected empty changes map after reset")
	}
	
	if len(stats.ErrorsByType) != 0 {
		t.Error("Expected empty errors map after reset")
	}
}

func TestMetrics_Concurrent(t *testing.T) {
	metrics := NewMetrics()
	
	// Параллельная запись
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			metrics.RecordConversion(10*time.Millisecond, 100, 90)
			metrics.RecordChange(ChangeUppercaseTag)
			done <- true
		}()
	}
	
	// Ждем завершения
	for i := 0; i < 100; i++ {
		<-done
	}
	
	stats := metrics.GetStats()
	
	if stats.TotalConversions != 100 {
		t.Errorf("Expected 100 conversions, got %d", stats.TotalConversions)
	}
	
	if stats.ChangesApplied[ChangeUppercaseTag] != 100 {
		t.Errorf("Expected 100 changes, got %d", stats.ChangesApplied[ChangeUppercaseTag])
	}
}

func TestNoOpMetrics(t *testing.T) {
	metrics := &NoOpMetrics{}
	
	// Не должно паниковать
	metrics.RecordConversion(100*time.Millisecond, 1000, 900)
	metrics.RecordError(ErrParseFailed)
	metrics.RecordChange(ChangeUppercaseTag)
	metrics.Reset()
	
	stats := metrics.GetStats()
	
	if stats.TotalConversions != 0 {
		t.Error("NoOpMetrics should return zero stats")
	}
}

func BenchmarkMetrics_RecordConversion(b *testing.B) {
	metrics := NewMetrics()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordConversion(10*time.Millisecond, 1000, 900)
	}
}

func BenchmarkMetrics_RecordChange(b *testing.B) {
	metrics := NewMetrics()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordChange(ChangeUppercaseTag)
	}
}
