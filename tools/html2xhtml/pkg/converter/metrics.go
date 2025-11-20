// pkg/converter/metrics.go
package converter

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics интерфейс для сбора метрик
type Metrics interface {
	RecordConversion(duration time.Duration, inputSize, outputSize int64)
	RecordError(errorType ErrorCode)
	RecordChange(changeType ChangeType)
	GetStats() ConversionStats
	Reset()
}

// ConversionStats статистика конвертаций
type ConversionStats struct {
	TotalConversions      int64
	SuccessfulConversions int64
	FailedConversions     int64
	AverageDuration       time.Duration
	TotalBytesProcessed   int64
	TotalBytesOutput      int64
	ChangesApplied        map[ChangeType]int64
	ErrorsByType          map[ErrorCode]int64
}

// DefaultMetrics реализация метрик
type DefaultMetrics struct {
	mu                    sync.RWMutex
	totalConversions      int64
	successfulConversions int64
	failedConversions     int64
	totalDuration         int64 // nanoseconds
	totalBytesProcessed   int64
	totalBytesOutput      int64
	changesApplied        map[ChangeType]int64
	errorsByType          map[ErrorCode]int64
}

func NewMetrics() Metrics {
	return &DefaultMetrics{
		changesApplied: make(map[ChangeType]int64),
		errorsByType:   make(map[ErrorCode]int64),
	}
}

func (m *DefaultMetrics) RecordConversion(duration time.Duration, inputSize, outputSize int64) {
	atomic.AddInt64(&m.totalConversions, 1)
	atomic.AddInt64(&m.successfulConversions, 1)
	atomic.AddInt64(&m.totalDuration, int64(duration))
	atomic.AddInt64(&m.totalBytesProcessed, inputSize)
	atomic.AddInt64(&m.totalBytesOutput, outputSize)
}

func (m *DefaultMetrics) RecordError(errorType ErrorCode) {
	atomic.AddInt64(&m.totalConversions, 1)
	atomic.AddInt64(&m.failedConversions, 1)
	
	m.mu.Lock()
	m.errorsByType[errorType]++
	m.mu.Unlock()
}

func (m *DefaultMetrics) RecordChange(changeType ChangeType) {
	m.mu.Lock()
	m.changesApplied[changeType]++
	m.mu.Unlock()
}

func (m *DefaultMetrics) GetStats() ConversionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := atomic.LoadInt64(&m.totalConversions)
	avgDuration := time.Duration(0)
	if total > 0 {
		avgDuration = time.Duration(atomic.LoadInt64(&m.totalDuration) / total)
	}

	// Копируем map'ы
	changes := make(map[ChangeType]int64)
	for k, v := range m.changesApplied {
		changes[k] = v
	}

	errors := make(map[ErrorCode]int64)
	for k, v := range m.errorsByType {
		errors[k] = v
	}

	return ConversionStats{
		TotalConversions:      atomic.LoadInt64(&m.totalConversions),
		SuccessfulConversions: atomic.LoadInt64(&m.successfulConversions),
		FailedConversions:     atomic.LoadInt64(&m.failedConversions),
		AverageDuration:       avgDuration,
		TotalBytesProcessed:   atomic.LoadInt64(&m.totalBytesProcessed),
		TotalBytesOutput:      atomic.LoadInt64(&m.totalBytesOutput),
		ChangesApplied:        changes,
		ErrorsByType:          errors,
	}
}

func (m *DefaultMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.StoreInt64(&m.totalConversions, 0)
	atomic.StoreInt64(&m.successfulConversions, 0)
	atomic.StoreInt64(&m.failedConversions, 0)
	atomic.StoreInt64(&m.totalDuration, 0)
	atomic.StoreInt64(&m.totalBytesProcessed, 0)
	atomic.StoreInt64(&m.totalBytesOutput, 0)

	m.changesApplied = make(map[ChangeType]int64)
	m.errorsByType = make(map[ErrorCode]int64)
}

// NoOpMetrics заглушка для отключения метрик
type NoOpMetrics struct{}

func (m *NoOpMetrics) RecordConversion(duration time.Duration, inputSize, outputSize int64) {}
func (m *NoOpMetrics) RecordError(errorType ErrorCode)                                      {}
func (m *NoOpMetrics) RecordChange(changeType ChangeType)                                   {}
func (m *NoOpMetrics) GetStats() ConversionStats {
	return ConversionStats{
		ChangesApplied: make(map[ChangeType]int64),
		ErrorsByType:   make(map[ErrorCode]int64),
	}
}
func (m *NoOpMetrics) Reset() {}
