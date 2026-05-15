package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Использование: %s <файл_с_исходным_кодом>\n", os.Args[0])
		os.Exit(1)
	}
	inputFile := os.Args[1]

	// ---------- 1. ПРЕПРОЦЕССИНГ (ЛР1) ----------
	fmt.Println("=== Препроцессинг ===")
	if err := Preprocess(inputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка препроцессинга: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Препроцессинг завершён. Результат в clean.cpp\n")

	// ---------- 2. ЛЕКСИЧЕСКИЙ АНАЛИЗ (ЛР2) ----------
	fmt.Println("=== Лексический анализ ===")
	cleanData, err := os.ReadFile("clean.cpp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка чтения clean.cpp: %v\n", err)
		os.Exit(1)
	}
	lexer := NewLexer(string(cleanData))
	lexer.Tokenize()
	if len(lexer.errors) > 0 {
		fmt.Println("Лексические ошибки:")
		for _, e := range lexer.errors {
			fmt.Printf("  - %s\n", e)
		}
		os.Exit(1)
	}
	fmt.Printf("Лексический анализ успешен. Найдено %d токенов.\n\n", len(lexer.tokens))

	// ---------- 3. СИНТАКСИЧЕСКИЙ АНАЛИЗ (ЛР3) ----------
	fmt.Println("=== Синтаксический анализ ===")
	parser := NewParser(lexer.tokens)
	ast, errs := parser.Parse()
	if len(errs) > 0 {
		fmt.Println("Синтаксические ошибки:")
		for _, e := range errs {
			fmt.Printf("  - %s\n", e)
		}
		os.Exit(1)
	}
	fmt.Println("Синтаксический анализ успешен. AST построено.")
	printAST(ast) // выводим дерево (опционально)
	fmt.Println()

	// ---------- 4. СЕМАНТИЧЕСКИЙ АНАЛИЗ И ГЕНЕРАЦИЯ ТРИАД (ЛР4) ----------
	fmt.Println("=== Семантический анализ и генерация триад ===")
	sem := NewSemanticAnalyzer()
	if !sem.Analyze(ast) {
		os.Exit(1)
	}
	fmt.Println("\nОбработка завершена успешно.")
}
