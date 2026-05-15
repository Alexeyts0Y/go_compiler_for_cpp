package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func Preprocess(inputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %v", err)
	}
	input := string(data)

	// 1. Проверка на недопустимые символы
	if err := checkInvalidChars(input); err != nil {
		return err
	}

	// 2. Проверка на незакрытые многострочные комментарии
	if err := checkUnclosedComments(input); err != nil {
		return err
	}

	// 3. Удаление комментариев
	cleaned := removeComments(input)

	// 4. Очистка пробельных символов и пустых строк
	result := cleanWhitespace(cleaned)

	// 5. Запись результата
	if err := os.WriteFile("clean.cpp", []byte(result), 0644); err != nil {
		return fmt.Errorf("ошибка записи clean.cpp: %v", err)
	}
	return nil
}

func checkInvalidChars(s string) error {
	for _, r := range s {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return fmt.Errorf("обнаружен недопустимый символ с кодом %d (U+%04X)", r, r)
		}
	}
	return nil
}

func checkUnclosedComments(s string) error {
	open := strings.Count(s, "/*")
	close := strings.Count(s, "*/")
	if open > close {
		return fmt.Errorf("незакрытый многострочный комментарий (найдено %d '/*' и %d '*/')", open, close)
	}
	if close > open {
		return fmt.Errorf("лишний символ закрытия комментария '*/' (найдено %d '/*' и %d '*/')", open, close)
	}

	balance := 0
	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
			balance++
			i++
		} else if i+1 < len(s) && s[i] == '*' && s[i+1] == '/' {
			if balance == 0 {
				return fmt.Errorf("закрывающая последовательность '*/' без открывающей '/*'")
			}
			balance--
			i++
		}
	}
	if balance != 0 {
		return fmt.Errorf("незакрытый многострочный комментарий (баланс %d)", balance)
	}
	return nil
}

func removeComments(s string) string {
	multiLineRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	singleLineRegex := regexp.MustCompile(`(?m)//.*$`)

	noMulti := multiLineRegex.ReplaceAllString(s, "")
	noComments := singleLineRegex.ReplaceAllString(noMulti, "")
	return noComments
}

func cleanWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	var resultLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		spaceRegex := regexp.MustCompile(`[ \t]+`)
		cleaned := spaceRegex.ReplaceAllString(trimmed, " ")
		resultLines = append(resultLines, cleaned)
	}
	return strings.Join(resultLines, "\n")
}
