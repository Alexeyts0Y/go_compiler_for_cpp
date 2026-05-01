package main

import (
	"fmt"
	"strings"
)

type TokenType string

const (
	KEYWORD      TokenType = "KEYWORD"
	IDENTIFIER   TokenType = "IDENTIFIER"
	CONSTANT_INT TokenType = "CONSTANT_INT"
	CONSTANT_STR TokenType = "CONSTANT_STR"
	OPERATOR     TokenType = "OPERATOR"
	DELIMITER    TokenType = "DELIMITER"
	UNKNOWN      TokenType = "UNKNOWN"
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
	Col   int
}

var keywords = map[string]bool{
	"using":     true,
	"namespace": true,
	"void":      true,
	"int":       true,
	"for":       true,
	"if":        true,
	"return":    true,
}

var operators = []string{"<<", "++", "=", "+", "-", "<", ">"}

// Разделители
var delimiters = map[rune]bool{
	';': true, ',': true, '{': true, '}': true,
	'(': true, ')': true, '[': true, ']': true,
}

// Лексер
type Lexer struct {
	input  string
	pos    int
	line   int
	col    int
	tokens []Token
	errors []string
}

func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		pos:    0,
		line:   1,
		col:    1,
		tokens: []Token{},
		errors: []string{},
	}
}

// Возвращает текущий символ (руну) или 0, если конец строки
func (l *Lexer) current() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	return rune(l.input[l.pos])
}

// Продвигает позицию на один символ, обновляет line/col
func (l *Lexer) advance() {
	if l.current() == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	l.pos++
}

// Пропуск пробельных символов (пробел, табуляция, перевод строки, возврат каретки)
func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		c := l.current()
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			l.advance()
		} else {
			break
		}
	}
}

func (l *Lexer) skipPreprocessorDirective() {
	for l.pos < len(l.input) && l.current() != '\n' {
		l.advance()
	}
	if l.current() == '\n' {
		l.advance()
	}
}

func isIdentifierLetter(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isDigit(c rune) bool {
	return c >= '0' && c <= '9'
}

// Чтение идентификатора или ключевого слова
func (l *Lexer) readIdentifier() string {
	start := l.pos
	for isIdentifierLetter(l.current()) || isDigit(l.current()) {
		l.advance()
	}
	return l.input[start:l.pos]
}

// Чтение целочисленной константы
func (l *Lexer) readInteger() (string, bool) {
	start := l.pos
	for isDigit(l.current()) {
		l.advance()
	}
	lexeme := l.input[start:l.pos]

	// Проверка на буквы после числа (некорректное число)
	if l.pos < len(l.input) && isIdentifierLetter(l.current()) {
		for isIdentifierLetter(l.current()) || isDigit(l.current()) {
			l.advance()
		}
		l.errors = append(l.errors, fmt.Sprintf("Некорректное число с буквами '%s' на строке %d, столбце %d",
			l.input[start:l.pos], l.line, l.col))
		return l.input[start:l.pos], false
	}

	// Проверка на точку (вещественное число не поддерживается)
	if l.pos < len(l.input) && l.current() == '.' {
		// Проверим, не является ли это частью вещественного числа
		next := l.peek()
		if isDigit(next) {
			l.errors = append(l.errors, fmt.Sprintf("Вещественные константы не поддерживаются (обнаружена точка в числе) на строке %d, столбце %d",
				l.line, l.col))
			// Проглатываем всё число с точкой, чтобы не зациклиться
			for isDigit(l.current()) || l.current() == '.' || l.current() == 'e' || l.current() == 'E' {
				l.advance()
			}
			return l.input[start:l.pos], false
		}
	}
	return lexeme, true
}

// Чтение строковой константы
func (l *Lexer) readString() (string, bool) {
	start := l.pos
	l.advance() // пропустить открывающую кавычку

	for {
		if l.pos >= len(l.input) {
			l.errors = append(l.errors, fmt.Sprintf("Незакрытая строковая константа на строке %d", l.line))
			return l.input[start:l.pos], false
		}
		c := l.current()
		if c == '"' {
			l.advance()
			break
		}
		if c == '\\' {
			l.advance() // пропустить обратный слеш
			if l.pos < len(l.input) {
				l.advance() // пропустить экранированный символ
			}
			continue
		}
		l.advance()
	}
	return l.input[start:l.pos], true
}

// Возвращает следующий символ без смещения позиции
func (l *Lexer) peek() rune {
	if l.pos+1 >= len(l.input) {
		return 0
	}
	return rune(l.input[l.pos+1])
}

// Попытка прочитать оператор из заранее заданного списка
func (l *Lexer) tryReadOperator() (string, bool) {
	for _, op := range operators {
		if strings.HasPrefix(l.input[l.pos:], op) {
			return op, true
		}
	}
	return "", false
}

// Основной метод разбора
func (l *Lexer) Tokenize() {
	for l.pos < len(l.input) {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}

		c := l.current()

		// Препроцессорная директива
		if c == '#' {
			l.skipPreprocessorDirective()
			continue
		}

		// Идентификатор или ключевое слово
		if isIdentifierLetter(c) {
			word := l.readIdentifier()
			if keywords[word] {
				l.tokens = append(l.tokens, Token{KEYWORD, word, l.line, l.col})
			} else {
				l.tokens = append(l.tokens, Token{IDENTIFIER, word, l.line, l.col})
			}
			continue
		}

		// Целочисленная константа
		if isDigit(c) {
			num, ok := l.readInteger()
			if !ok {
				l.tokens = append(l.tokens, Token{UNKNOWN, num, l.line, l.col})
			} else {
				l.tokens = append(l.tokens, Token{CONSTANT_INT, num, l.line, l.col})
			}
			continue
		}

		// Строковая константа
		if c == '"' {
			str, ok := l.readString()
			if ok {
				l.tokens = append(l.tokens, Token{CONSTANT_STR, str, l.line, l.col})
			} else {
				l.tokens = append(l.tokens, Token{UNKNOWN, str, l.line, l.col})
			}
			continue
		}

		// Оператор
		if op, found := l.tryReadOperator(); found {
			l.tokens = append(l.tokens, Token{OPERATOR, op, l.line, l.col})
			l.pos += len(op)
			continue
		}

		// Разделитель
		if delimiters[c] {
			l.tokens = append(l.tokens, Token{DELIMITER, string(c), l.line, l.col})
			l.advance()
			continue
		}

		// Недопустимый символ
		l.errors = append(l.errors, fmt.Sprintf("Недопустимый символ '%c' (U+%04X) на строке %d, столбце %d",
			c, c, l.line, l.col))
		l.tokens = append(l.tokens, Token{UNKNOWN, string(c), l.line, l.col})
		l.advance()
	}
}

// func main() {
// 	// Чтение очищенного файла
// 	data, err := os.ReadFile("clean.cpp")
// 	if err != nil {
// 		fmt.Fprintf(os.Stderr, "Ошибка чтения clean.cpp: %v\n", err)
// 		os.Exit(1)
// 	}
// 	source := string(data)

// 	// Лексический анализ
// 	lexer := NewLexer(source)
// 	lexer.Tokenize()

// 	// Вывод таблицы лексем
// 	fmt.Println("Лексема     | Тип")
// 	fmt.Println("------------+----------------------")
// 	for _, tok := range lexer.tokens {
// 		fmt.Printf("%-12s| %s\n", tok.Value, tok.Type)
// 	}

// 	// Формирование списка пар для синтаксического анализатора
// 	var tokenPairs []string
// 	for _, tok := range lexer.tokens {
// 		tokenPairs = append(tokenPairs, fmt.Sprintf("(%s, %s)", tok.Type, tok.Value))
// 	}
// 	fmt.Println()
// 	fmt.Println("[" + strings.Join(tokenPairs, ", ") + "]")

// 	// Итоги
// 	if len(lexer.errors) > 0 {
// 		fmt.Println("\nЛексические ошибки:")
// 		for _, errMsg := range lexer.errors {
// 			fmt.Println("  -", errMsg)
// 		}
// 		fmt.Printf("Анализ завершён с %d ошибками.\n", len(lexer.errors))
// 	} else {
// 		fmt.Printf("\nЛексический анализ завершён успешно. Обнаружено %d токенов. Ошибок не найдено.\n", len(lexer.tokens))
// 	}
// }
