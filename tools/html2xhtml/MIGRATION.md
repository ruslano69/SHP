# Миграция на v2

## Обратная совместимость

**Хорошая новость:** v2 полностью обратно совместим с v1!

Все старые методы работают как раньше:
```go
// v1 - работает в v2
conv := converter.New()
result, err := conv.Convert(input, opts)
```

## Что нужно обновить?

### 1. Middleware конфигурация (опционально)

Если хотите метрики:

**Было (v1):**
```go
config := middleware.Config{
    EnableCache: true,
}
```

**Стало (v2):**
```go
config := middleware.Config{
    EnableCache:   true,
    EnableMetrics: true, // новая опция
}
```

### 2. Использование контекста (рекомендуется)

**Было (v1):**
```go
result, err := conv.Convert(input, opts)
```

**Стало (v2):**
```go
ctx := r.Context() // из http.Request
result, err := conv.ConvertWithContext(ctx, input, opts)
```

### 3. Обработка ошибок (опционально)

**Было (v1):**
```go
if err != nil {
    log.Printf("Error: %v", err)
}
```

**Стало (v2):**
```go
if err != nil {
    if convErr, ok := err.(*converter.Error); ok {
        log.Printf("Error [%d]: %s - %v", 
            convErr.Code, convErr.Message, convErr.Cause)
    }
}
```

### 4. Метрики (опционально)

**Новое в v2:**
```go
// При создании
metrics := converter.NewMetrics()
conv := converter.NewWithMetrics(metrics)

// Получение статистики
stats := metrics.GetStats()
```

## Примеры миграции

### CLI - без изменений
```bash
# Работает как в v1
shp-convert -input ./site -output ./dist
```

### Middleware - минимальные изменения

**v1 код:**
```go
handler := middleware.XHTMLMiddleware(middleware.Config{
    EnableCache: true,
})(mux)
```

**v2 код (с метриками):**
```go
config := middleware.Config{
    EnableCache:   true,
    EnableMetrics: true,
}

handler := middleware.XHTMLMiddleware(config)(mux)

// Добавить endpoint метрик
mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
    json.NewEncoder(w).Encode(config.GetMetrics())
})
```

## Рекомендации

### Обязательно:
- ✅ Ничего! Код v1 работает без изменений

### Рекомендуется:
- 🔄 Использовать `ConvertWithContext()` для таймаутов
- 📊 Включить метрики в production

### Желательно:
- 🎯 Обрабатывать структурированные ошибки
- 📈 Добавить endpoint `/metrics` для мониторинга

## Тестирование после миграции

```bash
# Запустить все тесты
go test ./...

# С покрытием
go test -cover ./...

# Benchmark
go test -bench=. ./pkg/converter
```

## Troubleshooting

### Ошибка импорта
```
cannot find package "context"
```
**Решение:** Убедитесь что используете Go 1.21+

### Метрики не работают
```go
// Проверьте что флаг включен
config := middleware.Config{
    EnableMetrics: true, // <-- важно!
}
```

### Context timeout не срабатывает
```go
// Используйте правильный метод
result, err := conv.ConvertWithContext(ctx, input, opts) // ✅
// НЕ
result, err := conv.Convert(input, opts) // ❌ не поддерживает context
```
