package main

import (
	"fmt"
	"os"
	"strings"
)

// ---- AST-узлы ----
type ProgramNode struct {
	Using     *UsingDirectiveNode
	Functions []*FunctionDefNode // было *FunctionDefNode
}

type UsingDirectiveNode struct {
	Namespace string
}

type FunctionDefNode struct {
	ReturnType string
	Name       string
	Params     []ParamNode // список параметров
	Body       *CompoundStmtNode
}

type ParamNode struct {
	Type string
	Name string
}

type CompoundStmtNode struct {
	Statements []StmtNode
}

type StmtNode interface{ isStmt() }

type VarDeclNode struct {
	Type string
	Name string
	Init ExprNode // может быть nil
}

func (VarDeclNode) isStmt() {}

type AssignStmtNode struct {
	Left  ExprNode // теперь выражение, чтобы поддерживать arr[i] = ...
	Right ExprNode
}

func (AssignStmtNode) isStmt() {}

type IfStmtNode struct {
	Condition ExprNode
	Then      *CompoundStmtNode
	Else      *CompoundStmtNode
}

func (IfStmtNode) isStmt() {}

type ForStmtNode struct {
	Init      StmtNode // может быть VarDecl или AssignStmt (или nil)
	Condition ExprNode
	Post      ExprNode
	Body      *CompoundStmtNode
}

func (ForStmtNode) isStmt() {}

type ReturnStmtNode struct {
	Value ExprNode
}

func (ReturnStmtNode) isStmt() {}

type ExprStmtNode struct {
	Expr ExprNode
}

func (ExprStmtNode) isStmt() {}

// Выражения
type ExprNode interface{ isExpr() }

type BinaryExprNode struct {
	Op    string
	Left  ExprNode
	Right ExprNode
}

func (BinaryExprNode) isExpr() {}

type UnaryExprNode struct {
	Op     string // ++ постфиксный/префиксный упрощённо
	Right  ExprNode
	Prefix bool
}

func (UnaryExprNode) isExpr() {}

type IdentifierNode struct {
	Name string
}

func (IdentifierNode) isExpr() {}

type IntConstNode struct {
	Value string
}

func (IntConstNode) isExpr() {}

type IndexExprNode struct {
	Array ExprNode
	Index ExprNode
}

func (IndexExprNode) isExpr() {}

type CallExprNode struct {
	Callee ExprNode
	Args   []ExprNode
}

func (CallExprNode) isExpr() {}

// ---- Парсер ----
type Parser struct {
	tokens []Token
	pos    int
	errors []string
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, pos: 0, errors: []string{}}
}

func (p *Parser) peek() Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return Token{Type: "EOF", Value: "", Line: -1, Col: -1}
}

func (p *Parser) advance() { p.pos++ }

func (p *Parser) expect(typ TokenType, values ...string) (Token, error) {
	tok := p.peek()
	if tok.Type == "EOF" {
		return tok, fmt.Errorf("неожиданный конец файла, ожидался %s", p.desc(typ, values))
	}
	if tok.Type != typ {
		return tok, fmt.Errorf("ожидался %s, получен %s ('%s')", p.desc(typ, values), tok.Type, tok.Value)
	}
	if len(values) > 0 {
		found := false
		for _, v := range values {
			if tok.Value == v {
				found = true
				break
			}
		}
		if !found {
			return tok, fmt.Errorf("ожидался '%s', получен '%s'", strings.Join(values, "' или '"), tok.Value)
		}
	}
	p.advance()
	return tok, nil
}

func (p *Parser) desc(typ TokenType, values []string) string {
	if len(values) > 0 {
		return "'" + strings.Join(values, "' или '") + "'"
	}
	return string(typ)
}

func (p *Parser) addError(msg string, tok Token) {
	p.errors = append(p.errors, fmt.Sprintf("строка %d, столбец %d: %s", tok.Line, tok.Col, msg))
}

// ---- Главный разбор ----
func (p *Parser) Parse() (*ProgramNode, []string) {
	prog := &ProgramNode{}

	if p.peek().Type == KEYWORD && p.peek().Value == "using" {
		u, err := p.parseUsingDirective()
		if err != nil {
			p.addError(err.Error(), p.peek())
			return nil, p.errors
		}
		prog.Using = u
	}

	// Разбираем все функции, пока есть токены
	for p.pos < len(p.tokens) {
		fn, err := p.parseFunctionDef()
		if err != nil {
			p.addError(err.Error(), p.peek())
			return nil, p.errors
		}
		prog.Functions = append(prog.Functions, fn)
	}

	return prog, p.errors
}
func (p *Parser) parseUsingDirective() (*UsingDirectiveNode, error) {
	_, _ = p.expect(KEYWORD, "using")
	_, _ = p.expect(KEYWORD, "namespace")
	tok, err := p.expect(IDENTIFIER)
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(DELIMITER, ";"); err != nil {
		return nil, err
	}
	return &UsingDirectiveNode{Namespace: tok.Value}, nil
}

// Функция: тип имя ( параметры? ) тело
func (p *Parser) parseFunctionDef() (*FunctionDefNode, error) {
	typ, err := p.parseType()
	if err != nil {
		return nil, err
	}
	tok, err := p.expect(IDENTIFIER)
	if err != nil {
		return nil, err
	}
	name := tok.Value
	if _, err = p.expect(DELIMITER, "("); err != nil {
		return nil, err
	}
	// параметры
	var params []ParamNode
	if p.peek().Type != DELIMITER || p.peek().Value != ")" {
		params, err = p.parseParameterList()
		if err != nil {
			return nil, err
		}
	}
	if _, err = p.expect(DELIMITER, ")"); err != nil {
		return nil, err
	}
	body, err := p.parseCompoundStatement()
	if err != nil {
		return nil, err
	}
	return &FunctionDefNode{ReturnType: typ, Name: name, Params: params, Body: body}, nil
}

func (p *Parser) parseParameterList() ([]ParamNode, error) {
	var list []ParamNode
	for {
		param, err := p.parseParameter()
		if err != nil {
			return nil, err
		}
		list = append(list, param)
		if p.peek().Type == DELIMITER && p.peek().Value == "," {
			p.advance()
			continue
		}
		break
	}
	return list, nil
}

func (p *Parser) parseParameter() (ParamNode, error) {
	typ, err := p.parseType()
	if err != nil {
		return ParamNode{}, err
	}
	tok, err := p.expect(IDENTIFIER)
	if err != nil {
		return ParamNode{}, err
	}
	// разрешаем [] для параметров-массивов
	if p.peek().Type == DELIMITER && p.peek().Value == "[" {
		p.advance() // [
		if _, err = p.expect(DELIMITER, "]"); err != nil {
			return ParamNode{}, err
		}
	}
	return ParamNode{Type: typ, Name: tok.Value}, nil
}

func (p *Parser) parseType() (string, error) {
	tok := p.peek()
	if tok.Type == KEYWORD && (tok.Value == "int" || tok.Value == "void") {
		p.advance()
		return tok.Value, nil
	}
	return "", fmt.Errorf("ожидался тип, получен %s '%s'", tok.Type, tok.Value)
}

func (p *Parser) parseCompoundStatement() (*CompoundStmtNode, error) {
	if _, err := p.expect(DELIMITER, "{"); err != nil {
		return nil, err
	}
	node := &CompoundStmtNode{}
	for !(p.peek().Type == DELIMITER && p.peek().Value == "}") {
		if p.peek().Type == "EOF" {
			return nil, fmt.Errorf("незакрытый блок, ожидалась '}'")
		}
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		node.Statements = append(node.Statements, stmt)
	}
	p.advance() // }
	return node, nil
}

func (p *Parser) parseStatement() (StmtNode, error) {
	tok := p.peek()

	// Объявление переменной (int ...)
	if tok.Type == KEYWORD && (tok.Value == "int" || tok.Value == "void") {
		return p.parseVarDeclOrForInit()
	}
	if tok.Type == KEYWORD && tok.Value == "if" {
		return p.parseIfStmt()
	}
	if tok.Type == KEYWORD && tok.Value == "for" {
		return p.parseForStmt()
	}
	if tok.Type == KEYWORD && tok.Value == "return" {
		return p.parseReturnStmt()
	}
	// если начинается с идентификатора или константы/строки/скобки – оператор-выражение или присваивание
	// Попробуем разобрать как выражение, потом если ';', то ExprStmt, если '=' то AssignStmt
	// нужно отличать присваивание (a = expr;) от выражения-оператора (cout << ...;)
	// Упростим: разбираем выражение, потом смотрим, что дальше:
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	if p.peek().Type == OPERATOR && p.peek().Value == "=" {
		// присваивание: левая часть уже разобрана, ожидаем = и выражение справа
		p.advance() // =
		right, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err = p.expect(DELIMITER, ";"); err != nil {
			return nil, err
		}
		return &AssignStmtNode{Left: expr, Right: right}, nil
	}
	// иначе это просто выражение-оператор
	if _, err = p.expect(DELIMITER, ";"); err != nil {
		return nil, err
	}
	return &ExprStmtNode{Expr: expr}, nil
}

func (p *Parser) parseVarDeclOrForInit() (StmtNode, error) {
	typ, err := p.parseType()
	if err != nil {
		return nil, err
	}
	tok, err := p.expect(IDENTIFIER)
	if err != nil {
		return nil, err
	}
	name := tok.Value
	// Может быть объявление массива: int arr[10] = {...} или int arr[]
	if p.peek().Type == DELIMITER && p.peek().Value == "[" {
		p.advance()
		// размер массива (может быть число, или пусто)
		if p.peek().Type == CONSTANT_INT || p.peek().Type == IDENTIFIER {
			p.advance() // пропускаем размер
		} // иначе просто ]
		if _, err = p.expect(DELIMITER, "]"); err != nil {
			return nil, err
		}
	}
	// инициализация (необязательно)
	var init ExprNode
	if p.peek().Type == OPERATOR && p.peek().Value == "=" {
		p.advance()
		// если следующая '{' - список инициализации, обработаем упрощённо
		if p.peek().Type == DELIMITER && p.peek().Value == "{" {
			// пропускаем содержимое до закрывающей '}'
			p.skipBalancedBraces()
		} else {
			init, err = p.parseExpression()
			if err != nil {
				return nil, err
			}
		}
	}
	if _, err = p.expect(DELIMITER, ";"); err != nil {
		return nil, err
	}
	return &VarDeclNode{Type: typ, Name: name, Init: init}, nil
}

// skipBalancedBraces пропускает всё от текущей { до соответствующей }, поддерживая вложенность
func (p *Parser) skipBalancedBraces() {
	if p.peek().Type != DELIMITER || p.peek().Value != "{" {
		return
	}
	depth := 0
	for p.pos < len(p.tokens) {
		tok := p.peek()
		if tok.Type == DELIMITER {
			if tok.Value == "{" {
				depth++
			} else if tok.Value == "}" {
				depth--
				if depth == 0 {
					p.advance() // пропустить }
					return
				}
			}
		}
		p.advance()
	}
}

func (p *Parser) parseIfStmt() (*IfStmtNode, error) {
	_, _ = p.expect(KEYWORD, "if")
	_, _ = p.expect(DELIMITER, "(")
	cond, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	_, _ = p.expect(DELIMITER, ")")
	then, err := p.parseCompoundStatement()
	if err != nil {
		return nil, err
	}
	var elseBody *CompoundStmtNode
	if p.peek().Type == KEYWORD && p.peek().Value == "else" {
		p.advance()
		elseBody, err = p.parseCompoundStatement()
		if err != nil {
			return nil, err
		}
	}
	return &IfStmtNode{Condition: cond, Then: then, Else: elseBody}, nil
}

func (p *Parser) parseForStmt() (*ForStmtNode, error) {
	_, _ = p.expect(KEYWORD, "for")
	_, _ = p.expect(DELIMITER, "(")
	// init
	var init StmtNode
	if p.peek().Type == KEYWORD && (p.peek().Value == "int" || p.peek().Value == "void") {
		// объявление переменной
		init, _ = p.parseVarDeclOrForInit()
	} else if !(p.peek().Type == DELIMITER && p.peek().Value == ";") {
		// может быть присваивание или выражение
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if p.peek().Type == OPERATOR && p.peek().Value == "=" {
			p.advance()
			right, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			init = &AssignStmtNode{Left: expr, Right: right}
		} else {
			init = &ExprStmtNode{Expr: expr}
		}
	}
	_, _ = p.expect(DELIMITER, ";")
	// condition
	var cond ExprNode
	if !(p.peek().Type == DELIMITER && p.peek().Value == ";") {
		cond, _ = p.parseExpression()
	}
	_, _ = p.expect(DELIMITER, ";")
	// post
	var post ExprNode
	if !(p.peek().Type == DELIMITER && p.peek().Value == ")") {
		post, _ = p.parseExpression()
	}
	_, _ = p.expect(DELIMITER, ")")
	body, err := p.parseCompoundStatement()
	if err != nil {
		return nil, err
	}
	return &ForStmtNode{Init: init, Condition: cond, Post: post, Body: body}, nil
}

func (p *Parser) parseReturnStmt() (*ReturnStmtNode, error) {
	_, _ = p.expect(KEYWORD, "return")
	var expr ExprNode
	if !(p.peek().Type == DELIMITER && p.peek().Value == ";") {
		expr, _ = p.parseExpression()
	}
	_, _ = p.expect(DELIMITER, ";")
	return &ReturnStmtNode{Value: expr}, nil
}

// ---- Выражения ----

// Expression ::= Assignment
// Assignment ::= LogicalOr ('=' LogicalOr)?  -- мы вынесли присваивание на уровень statement, поэтому здесь не нужно
// Но для упрощения будем разбирать от низкого приоритета: сложение, сравнение, битовые сдвиги (<<)
func (p *Parser) parseExpression() (ExprNode, error) {
	return p.parseComparison()
}

// shift: addition ( ('<<'|'>>') addition )*
func (p *Parser) parseShift() (ExprNode, error) {
	left, err := p.parseAddition()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == OPERATOR && (p.peek().Value == "<<" || p.peek().Value == ">>") {
		op := p.peek().Value
		p.advance()
		right, err := p.parseAddition()
		if err != nil {
			return nil, err
		}
		left = &BinaryExprNode{Op: op, Left: left, Right: right}
	}
	return left, nil
}

// addition: comparison ( ('+'|'-') comparison )*
func (p *Parser) parseAddition() (ExprNode, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == OPERATOR && (p.peek().Value == "+" || p.peek().Value == "-") {
		op := p.peek().Value
		p.advance()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = &BinaryExprNode{Op: op, Left: left, Right: right}
	}
	return left, nil
}

// comparison: unary ( ('<'|'>'|'<='|'>='|'=='|'!=') unary )?  -- у нас есть <, >
func (p *Parser) parseComparison() (ExprNode, error) {
	left, err := p.parseShift()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == OPERATOR && (p.peek().Value == "<" || p.peek().Value == ">" || p.peek().Value == "<=" || p.peek().Value == ">=") {
		op := p.peek().Value
		p.advance()
		right, err := p.parseShift()
		if err != nil {
			return nil, err
		}
		left = &BinaryExprNode{Op: op, Left: left, Right: right}
	}
	return left, nil
}

// unary: ('++'|'--')? postfix
func (p *Parser) parseUnary() (ExprNode, error) {
	if p.peek().Type == OPERATOR && (p.peek().Value == "++" || p.peek().Value == "--") {
		op := p.peek().Value
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExprNode{Op: op, Right: operand, Prefix: true}, nil
	}
	return p.parsePostfix()
}

// postfix: primary ( '[' expr ']' | '(' args? ')' | '++' | '--' )*
func (p *Parser) parsePostfix() (ExprNode, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.peek()
		if tok.Type == DELIMITER && tok.Value == "[" {
			p.advance()
			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			_, err = p.expect(DELIMITER, "]")
			if err != nil {
				return nil, err
			}
			expr = &IndexExprNode{Array: expr, Index: index}
		} else if tok.Type == DELIMITER && tok.Value == "(" {
			p.advance()
			var args []ExprNode
			if !(p.peek().Type == DELIMITER && p.peek().Value == ")") {
				args, err = p.parseArguments()
				if err != nil {
					return nil, err
				}
			}
			_, err = p.expect(DELIMITER, ")")
			if err != nil {
				return nil, err
			}
			expr = &CallExprNode{Callee: expr, Args: args}
		} else if tok.Type == OPERATOR && (tok.Value == "++" || tok.Value == "--") {
			op := tok.Value
			p.advance()
			expr = &UnaryExprNode{Op: op, Right: expr, Prefix: false}
		} else {
			break
		}
	}
	return expr, nil
}

func (p *Parser) parseArguments() ([]ExprNode, error) {
	var args []ExprNode
	for {
		arg, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if p.peek().Type == DELIMITER && p.peek().Value == "," {
			p.advance()
			continue
		}
		break
	}
	return args, nil
}

// primary: IDENTIFIER | CONSTANT_INT | CONSTANT_STR | '(' expression ')'
func (p *Parser) parsePrimary() (ExprNode, error) {
	tok := p.peek()
	switch tok.Type {
	case IDENTIFIER:
		p.advance()
		return &IdentifierNode{Name: tok.Value}, nil
	case CONSTANT_INT:
		p.advance()
		return &IntConstNode{Value: tok.Value}, nil
	case CONSTANT_STR:
		p.advance()
		return &IdentifierNode{Name: tok.Value}, nil // строку обработаем как идентификатор для упрощения
	case DELIMITER:
		if tok.Value == "(" {
			p.advance()
			expr, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			if _, err = p.expect(DELIMITER, ")"); err != nil {
				return nil, err
			}
			return expr, nil
		}
	}
	return nil, fmt.Errorf("ожидался идентификатор, число или '(', получен %s '%s'", tok.Type, tok.Value)
}

// ---- Вывод AST ----
func printAST(prog *ProgramNode) {
	fmt.Println("Program")
	if prog.Using != nil {
		fmt.Println("├── using")
		fmt.Printf("│   └── namespace: %s\n", prog.Using.Namespace)
	}
	fmt.Println("└── functions")
	for i, f := range prog.Functions {
		lastFunc := i == len(prog.Functions)-1
		printFunction(f, "    ", lastFunc)
	}
}

func printFunction(f *FunctionDefNode, indent string, last bool) {
	branch := "├── "
	childIndent := indent + "│   "
	if last {
		branch = "└── "
		childIndent = indent + "    "
	}
	fmt.Printf("%s%sfunction\n", indent, branch)
	fmt.Printf("%s├── return_type: %s\n", childIndent, f.ReturnType)
	fmt.Printf("%s├── name: %s\n", childIndent, f.Name)
	fmt.Printf("%s├── params: [", childIndent)
	for j, p := range f.Params {
		if j > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%s %s", p.Type, p.Name)
	}
	fmt.Println("]")
	fmt.Printf("%s└── body:\n", childIndent)
	printCompoundStmt(f.Body, childIndent+"    ")
}

func printCompoundStmt(cs *CompoundStmtNode, indent string) {
	for i, stmt := range cs.Statements {
		isLast := i == len(cs.Statements)-1
		branch := "├── "
		childIndent := indent + "│   "
		if isLast {
			branch = "└── "
			childIndent = indent + "    "
		}
		fmt.Print(indent + branch)
		printStmt(stmt, childIndent, isLast)
	}
}

func printStmt(stmt StmtNode, childIndent string, last bool) {
	switch s := stmt.(type) {
	case *VarDeclNode:
		fmt.Printf("var_decl: %s %s", s.Type, s.Name)
		if s.Init != nil {
			fmt.Print(" = ")
			printExpr(s.Init)
		}
		fmt.Println()
	case *AssignStmtNode:
		fmt.Println("assign_stmt")
		fmt.Printf("%s├── left: ", childIndent)
		printExpr(s.Left)
		fmt.Println()
		fmt.Printf("%s└── right: ", childIndent)
		printExpr(s.Right)
		fmt.Println()
	case *IfStmtNode:
		fmt.Println("if_stmt")
		fmt.Printf("%s├── condition: ", childIndent)
		printExpr(s.Condition)
		fmt.Println()
		if s.Else != nil {
			fmt.Printf("%s├── then:\n", childIndent)
			printCompoundStmt(s.Then, childIndent+"│   ")
			fmt.Printf("%s└── else:\n", childIndent)
			printCompoundStmt(s.Else, childIndent+"    ")
		} else {
			fmt.Printf("%s└── then:\n", childIndent)
			printCompoundStmt(s.Then, childIndent+"    ")
		}
	case *ForStmtNode:
		fmt.Println("for_stmt")
		fmt.Printf("%s├── init: ", childIndent)
		if s.Init != nil {
			printStmtInline(s.Init)
		}
		fmt.Println()
		fmt.Printf("%s├── condition: ", childIndent)
		if s.Condition != nil {
			printExpr(s.Condition)
		}
		fmt.Println()
		fmt.Printf("%s├── post: ", childIndent)
		if s.Post != nil {
			printExpr(s.Post)
		}
		fmt.Println()
		fmt.Printf("%s└── body:\n", childIndent)
		printCompoundStmt(s.Body, childIndent+"    ")
	case *ReturnStmtNode:
		fmt.Print("return_stmt")
		if s.Value != nil {
			fmt.Print(": ")
			printExpr(s.Value)
		}
		fmt.Println()
	case *ExprStmtNode:
		fmt.Print("expr_stmt: ")
		printExpr(s.Expr)
		fmt.Println()
	default:
		// На случай, если встретился неизвестный тип утверждения
		fmt.Printf("unknown_stmt(%T)\n", stmt)
	}
}

func printStmtInline(stmt StmtNode) {
	switch s := stmt.(type) {
	case *VarDeclNode:
		fmt.Printf("var_decl %s %s", s.Type, s.Name)
	case *AssignStmtNode:
		printExpr(s.Left)
		fmt.Print(" = ")
		printExpr(s.Right)
	case *ExprStmtNode:
		printExpr(s.Expr)
	default:
		fmt.Printf("?")
	}
}

func printExpr(expr ExprNode) {
	switch e := expr.(type) {
	case *IdentifierNode:
		fmt.Print(e.Name)
	case *IntConstNode:
		fmt.Print(e.Value)
	case *BinaryExprNode:
		fmt.Print("(")
		printExpr(e.Left)
		fmt.Printf(" %s ", e.Op)
		printExpr(e.Right)
		fmt.Print(")")
	case *UnaryExprNode:
		if e.Prefix {
			fmt.Printf("%s", e.Op)
			printExpr(e.Right)
		} else {
			printExpr(e.Right)
			fmt.Printf("%s", e.Op)
		}
	case *IndexExprNode:
		printExpr(e.Array)
		fmt.Print("[")
		printExpr(e.Index)
		fmt.Print("]")
	case *CallExprNode:
		printExpr(e.Callee)
		fmt.Print("(")
		for i, a := range e.Args {
			if i > 0 {
				fmt.Print(", ")
			}
			printExpr(a)
		}
		fmt.Print(")")
	default:
		fmt.Print("?")
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Использование: go run lexer.go parser.go clean.cpp")
		os.Exit(1)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка чтения: %v\n", err)
		os.Exit(1)
	}
	source := string(data)

	lexer := NewLexer(source)
	lexer.Tokenize()
	if len(lexer.errors) > 0 {
		fmt.Println("Лексические ошибки:")
		for _, e := range lexer.errors {
			fmt.Println("  -", e)
		}
		os.Exit(1)
	}

	parser := NewParser(lexer.tokens)
	ast, errs := parser.Parse()
	if len(errs) > 0 {
		fmt.Println("\nСинтаксические ошибки:")
		for _, e := range errs {
			fmt.Println("  -", e)
		}
		fmt.Printf("Анализ завершён с %d ошибками.\n", len(errs))
		os.Exit(1)
	}

	fmt.Println("Синтаксический анализ завершён успешно. AST:")
	printAST(ast)
	fmt.Println("\nОшибок не найдено.")
}
