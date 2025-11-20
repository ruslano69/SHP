# SHP Converter - HTML to XHTML

Библиотека и утилита для конвертации HTML в XHTML с поддержкой SHP (Signed Hypertext Protocol).

⚠️ **Статус:** Research proposal - не для production

## Установка

```bash
go get github.com/ruslano69/shp
```

## CLI Утилита

### Установка
```bash
go install github.com/ruslano69/shp/cmd/shp-convert@latest
```

### Использование
```bash
# Базовая конвертация
shp-convert -input ./site -output ./dist

# Только валидация
shp-convert -input ./site -validate-only

# Строгий режим с детальным выводом
shp-convert -input ./site -output ./dist -strict -verbose

# Без автоисправления
shp-convert -input ./site -output ./dist -fix=false

# Без рекурсии в подпапки
shp-convert -input ./site -recursive=false
```

### Флаги
- `-input` - входная директория (default: ".")
- `-output` - выходная директория (default: "./dist")
- `-strict` - строгий режим, остановка при ошибках (default: false)
- `-fix` - автоисправление ошибок (default: true)
- `-verbose` - детальный вывод (default: false)
- `-validate-only` - только валидация без конвертации (default: false)
- `-recursive` - обработка поддиректорий (default: true)

## Middleware для фреймворков

### net/http
```go
import (
    "github.com/ruslano69/shp/middleware"
    "github.com/ruslano69/shp/pkg/converter"
)

mux := http.NewServeMux()
mux.HandleFunc("/", yourHandler)

config := middleware.Config{
    EnableCache:   true,
    EnableMetrics: true, // Включить метрики
    Options: converter.Options{
        AutoFix: true,
    },
    SkipPaths: []string{"/api/", "/static/"},
}

wrapped := middleware.XHTMLMiddleware(config)(mux)

// Endpoint для метрик
mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
    stats := config.GetMetrics()
    json.NewEncoder(w).Encode(stats)
})

http.ListenAndServe(":8080", wrapped)
```

### Gin
```go
import "github.com/ruslano69/shp/middleware"

router := gin.Default()

router.Use(middleware.GinMiddleware(middleware.Config{
    EnableCache: true,
    Options: converter.Options{
        AutoFix: true,
    },
}))

router.GET("/", yourHandler)
router.Run(":8080")
```

### Echo
```go
import "github.com/ruslano69/shp/middleware"

e := echo.New()

e.Use(middleware.EchoMiddleware(middleware.Config{
    EnableCache: true,
    Options: converter.Options{
        AutoFix: true,
    },
}))

e.GET("/", yourHandler)
e.Start(":8080")
```

## Конфигурация

### converter.Options
```go
type Options struct {
    StrictMode         bool // Строгий режим - отказ при ошибках
    AutoFix            bool // Автоисправление ошибок
    Verbose            bool // Детальные логи
    PreserveFormatting bool // Сохранять форматирование
    ValidateOnly       bool // Только валидация
}
```

### middleware.Config
```go
type Config struct {
    Converter      converter.Converter // Кастомный конвертер
    Options        converter.Options   // Опции конвертации
    EnableCache    bool                // Включить кеш
    EnableMetrics  bool                // Включить метрики
    SkipPaths      []string            // Пропустить пути
    OnlyExtensions []string            // Обрабатывать только эти расширения
}
```

## Новые возможности (v2)

### Context Support
Все операции теперь поддерживают контекст для:
- Таймаутов
- Отмены операций
- Передачи метаданных

### Структурированные ошибки
```go
type Error struct {
    Code    ErrorCode              // Код ошибки
    Message string                 // Сообщение
    Cause   error                  // Причина
    Field   string                 // Поле с ошибкой
    Context map[string]interface{} // Дополнительный контекст
}

// Коды ошибок
const (
    ErrParseFailed       // Ошибка парсинга
    ErrValidationFailed  // Ошибка валидации
    ErrConversionFailed  // Ошибка конвертации
    ErrTimeout           // Таймаут
    ErrContextCanceled   // Контекст отменен
    ErrInvalidInput      // Неверный ввод
)
```

### Метрики
Автоматический сбор статистики:
- Количество конвертаций (успешных/неуспешных)
- Средняя длительность
- Объем обработанных данных
- Типы внесенных изменений
- Типы ошибок

```go
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
```

## Использование библиотеки

```go
import "github.com/ruslano69/shp/pkg/converter"

conv := converter.New()

// Конвертация
result, err := conv.Convert(htmlBytes, converter.Options{
    AutoFix: true,
})

if err != nil {
    log.Fatal(err)
}

fmt.Printf("Success: %v\n", result.Success)
fmt.Printf("Changes: %d\n", len(result.Changes))
fmt.Printf("Output size: %d bytes\n", result.FinalSize)

// Только валидация
err = conv.Validate(htmlBytes)
if err != nil {
    log.Printf("Invalid XHTML: %v", err)
}
```

### С поддержкой контекста

```go
import (
    "context"
    "time"
    "github.com/ruslano69/shp/pkg/converter"
)

conv := converter.New()
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// Конвертация с таймаутом
result, err := conv.ConvertWithContext(ctx, htmlBytes, converter.Options{
    AutoFix: true,
})

if err != nil {
    if convErr, ok := err.(*converter.Error); ok {
        fmt.Printf("Error code: %d, message: %s\n", convErr.Code, convErr.Message)
    }
}
```

### С метриками

```go
import "github.com/ruslano69/shp/pkg/converter"

// Создаем конвертер с метриками
metrics := converter.NewMetrics()
conv := converter.NewWithMetrics(metrics)

// Выполняем конвертации
result, _ := conv.ConvertWithContext(ctx, htmlBytes, opts)

// Получаем статистику
stats := metrics.GetStats()
fmt.Printf("Total conversions: %d\n", stats.TotalConversions)
fmt.Printf("Success rate: %.2f%%\n", 
    float64(stats.SuccessfulConversions)/float64(stats.TotalConversions)*100)
fmt.Printf("Average duration: %v\n", stats.AverageDuration)
```

## Автоисправления

Конвертер автоматически исправляет:

- ✅ Unclosed void elements: `<br>` → `<br />`
- ✅ Uppercase tags: `<DIV>` → `<div>`
- ✅ Uppercase attributes: `CLASS="test"` → `class="test"`
- ✅ Unquoted attributes: `width=100` → `width="100"`
- ✅ Special characters: `&` → `&amp;`, `<` → `&lt;`

## Тестирование

```bash
# Запуск тестов
cd pkg/converter
go test -v

# Бенчмарки
go test -bench=.

# С покрытием
go test -cover
```

## Структура проекта

```
shp/
├── pkg/
│   └── converter/          # Ядро библиотеки
│       ├── converter.go
│       └── converter_test.go
├── cmd/
│   └── shp-convert/        # CLI утилита
│       └── main.go
├── middleware/             # Адаптеры для фреймворков
│   ├── http.go            # net/http
│   ├── gin.go             # Gin
│   └── echo.go            # Echo
├── docs/                   # Документация
│   ├── PROJECT_README.md
│   └── SPECIFICATION.md
└── examples/               # Примеры
    ├── demo.html
    └── shp-verify.js
```

## Документация

- [Полная спецификация SHP](docs/SPECIFICATION.md)
- [Описание проекта](docs/PROJECT_README.md)
- [Примеры использования](examples/)

## Лицензия

MIT License - See LICENSE file

## Автор

Ruslan - Ukraine 🇺🇦

## Контакты

- GitHub Issues: https://github.com/ruslano69/shp/issues
- Email: contact@shp-protocol.org
