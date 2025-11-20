// cmd/shp-convert/main.go
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ruslano69/shp/tools/html2xhtml/pkg/converter"
)

type Stats struct {
	TotalFiles    int
	SuccessCount  int
	FailedCount   int
	TotalChanges  int
	TotalErrors   int
	TotalSize     int64
	ProcessedSize int64
	StartTime     time.Time
}

func main() {
	// Флаги
	inputDir := flag.String("input", ".", "Input directory with HTML files")
	outputDir := flag.String("output", "./dist", "Output directory for XHTML")
	strict := flag.Bool("strict", false, "Strict mode: fail on any error")
	fix := flag.Bool("fix", true, "Auto-fix common errors")
	verbose := flag.Bool("verbose", false, "Verbose output")
	validateOnly := flag.Bool("validate-only", false, "Only validate, don't convert")
	recursive := flag.Bool("recursive", true, "Process subdirectories")
	
	flag.Parse()

	// Инициализация
	conv := converter.New()
	stats := &Stats{StartTime: time.Now()}

	fmt.Printf("🔧 SHP HTML→XHTML Converter\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Input:  %s\n", *inputDir)
	if !*validateOnly {
		fmt.Printf("Output: %s\n", *outputDir)
	}
	fmt.Printf("Mode:   ")
	if *strict {
		fmt.Printf("strict ")
	}
	if *fix {
		fmt.Printf("auto-fix ")
	}
	if *validateOnly {
		fmt.Printf("validate-only")
	}
	fmt.Printf("\n\n")

	// Создание output директории
	if !*validateOnly {
		if err := os.MkdirAll(*outputDir, 0755); err != nil {
			fmt.Printf("❌ Failed to create output directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Обработка файлов
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if !*recursive && path != *inputDir {
				return filepath.SkipDir
			}
			return nil
		}

		// Только HTML файлы
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".html") {
			return nil
		}

		stats.TotalFiles++
		stats.TotalSize += info.Size()

		return processFile(path, *inputDir, *outputDir, conv, stats, converter.Options{
			StrictMode:   *strict,
			AutoFix:      *fix,
			Verbose:      *verbose,
			ValidateOnly: *validateOnly,
		})
	}

	err := filepath.Walk(*inputDir, walkFunc)
	if err != nil {
		fmt.Printf("❌ Walk error: %v\n", err)
		os.Exit(1)
	}

	// Итоговый отчет
	printReport(stats)

	if stats.FailedCount > 0 && *strict {
		os.Exit(1)
	}
}

func processFile(path, inputDir, outputDir string, conv converter.Converter, stats *Stats, opts converter.Options) error {
	// Чтение
	content, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("❌ Error reading %s: %v\n", path, err)
		stats.FailedCount++
		return nil
	}

	relPath, _ := filepath.Rel(inputDir, path)

	// Валидация или конвертация
	if opts.ValidateOnly {
		err := conv.Validate(content)
		if err != nil {
			fmt.Printf("❌ %s: %v\n", relPath, err)
			stats.FailedCount++
			stats.TotalErrors++
		} else {
			fmt.Printf("✅ %s\n", relPath)
			stats.SuccessCount++
		}
		return nil
	}

	// Конвертация
	result, err := conv.Convert(content, opts)
	if err != nil {
		fmt.Printf("❌ %s: conversion failed: %v\n", relPath, err)
		stats.FailedCount++
		stats.TotalErrors++
		return nil
	}

	if !result.Success && opts.StrictMode {
		fmt.Printf("❌ %s: validation failed\n", relPath)
		for _, e := range result.Errors {
			fmt.Printf("   • %v\n", e)
		}
		stats.FailedCount++
		stats.TotalErrors += len(result.Errors)
		return nil
	}

	// Запись
	outPath := filepath.Join(outputDir, relPath)
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		fmt.Printf("❌ Failed to create directory for %s\n", outPath)
		stats.FailedCount++
		return nil
	}

	if err := ioutil.WriteFile(outPath, result.Output, 0644); err != nil {
		fmt.Printf("❌ Error writing %s: %v\n", outPath, err)
		stats.FailedCount++
		return nil
	}

	// Статистика
	stats.SuccessCount++
	stats.TotalChanges += len(result.Changes)
	stats.TotalErrors += len(result.Errors)
	stats.ProcessedSize += result.FinalSize

	// Вывод
	if opts.Verbose {
		fmt.Printf("✅ %s (%d changes", relPath, len(result.Changes))
		if len(result.Errors) > 0 {
			fmt.Printf(", %d warnings", len(result.Errors))
		}
		fmt.Printf(")\n")
		for _, change := range result.Changes {
			fmt.Printf("   • %s: %s → %s\n", change.Message, change.Original, change.Fixed)
		}
	} else {
		icon := "✅"
		if len(result.Errors) > 0 {
			icon = "⚠️"
		}
		fmt.Printf("%s %s", icon, relPath)
		if len(result.Changes) > 0 {
			fmt.Printf(" (%d fixes)", len(result.Changes))
		}
		fmt.Printf("\n")
	}

	return nil
}

func printReport(stats *Stats) {
	duration := time.Since(stats.StartTime)
	
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📊 Conversion Report\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Duration:     %v\n", duration.Round(time.Millisecond))
	fmt.Printf("Total files:  %d\n", stats.TotalFiles)
	fmt.Printf("Success:      %d\n", stats.SuccessCount)
	fmt.Printf("Failed:       %d\n", stats.FailedCount)
	fmt.Printf("Changes made: %d\n", stats.TotalChanges)
	
	if stats.TotalErrors > 0 {
		fmt.Printf("Errors:       %d\n", stats.TotalErrors)
	}
	
	if stats.ProcessedSize > 0 {
		fmt.Printf("Input size:   %.2f KB\n", float64(stats.TotalSize)/1024)
		fmt.Printf("Output size:  %.2f KB\n", float64(stats.ProcessedSize)/1024)
		
		ratio := float64(stats.ProcessedSize) / float64(stats.TotalSize) * 100
		fmt.Printf("Size ratio:   %.1f%%\n", ratio)
	}

	if stats.SuccessCount == stats.TotalFiles && stats.TotalFiles > 0 {
		fmt.Printf("\n🎉 All files converted successfully!\n")
	} else if stats.FailedCount > 0 {
		fmt.Printf("\n⚠️  %d files failed conversion\n", stats.FailedCount)
	}
}
