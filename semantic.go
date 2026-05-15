package main

import (
	"fmt"
)

// ---------- Таблица символов ----------
type Symbol struct {
	Name        string
	Type        string
	Declared    bool
	Initialized bool
	Scope       string
	Line        int
	IsArray     bool
	ArraySize   int
	IsFunction  bool
}

type SymbolTable struct {
	symbols map[string]*Symbol
	parent  *SymbolTable
	scope   string
}

func NewSymbolTable(scope string, parent *SymbolTable) *SymbolTable {
	return &SymbolTable{
		symbols: make(map[string]*Symbol),
		parent:  parent,
		scope:   scope,
	}
}

func (st *SymbolTable) Define(name, typ string, line int, initialized, isArray bool, size int) error {
	if _, ok := st.symbols[name]; ok {
		return fmt.Errorf("строка %d: повторное объявление '%s' в области '%s'", line, name, st.scope)
	}
	st.symbols[name] = &Symbol{
		Name:        name,
		Type:        typ,
		Declared:    true,
		Initialized: initialized,
		Scope:       st.scope,
		Line:        line,
		IsArray:     isArray,
		ArraySize:   size,
	}
	return nil
}

func (st *SymbolTable) DefineFunction(name, typ string) {
	st.symbols[name] = &Symbol{
		Name:        name,
		Type:        typ,
		Declared:    true,
		Initialized: true,
		Scope:       st.scope,
		IsFunction:  true,
	}
}

func (st *SymbolTable) Lookup(name string) (*Symbol, bool) {
	if sym, ok := st.symbols[name]; ok {
		return sym, true
	}
	if st.parent != nil {
		return st.parent.Lookup(name)
	}
	return nil, false
}

// ---------- Триады ----------
type Triad struct {
	Op   string
	Arg1 string
	Arg2 string
}

func (t Triad) String() string {
	return fmt.Sprintf("(%s, %s, %s)", t.Op, t.Arg1, t.Arg2)
}

// ---------- Семантический анализатор ----------
type SemanticAnalyzer struct {
	curTable    *SymbolTable
	globalTable *SymbolTable
	allTables   []*SymbolTable
	funcRetType string
	triads      []Triad
	errors      []string
	tempCounter int
}

func NewSemanticAnalyzer() *SemanticAnalyzer {
	sa := &SemanticAnalyzer{
		triads:      []Triad{},
		errors:      []string{},
		tempCounter: 0,
		allTables:   []*SymbolTable{},
	}
	sa.globalTable = NewSymbolTable("global", nil)
	sa.curTable = sa.globalTable
	sa.allTables = append(sa.allTables, sa.globalTable)

	// Предопределённые объекты cout и endl
	sa.globalTable.Define("cout", "ostream", 0, true, false, 0)
	sa.globalTable.Define("endl", "ostream", 0, true, false, 0)

	return sa
}

func (sa *SemanticAnalyzer) newTemp() string {
	sa.tempCounter++
	return fmt.Sprintf("_t%d", sa.tempCounter)
}

func (sa *SemanticAnalyzer) addTriad(op, arg1, arg2 string) int {
	sa.triads = append(sa.triads, Triad{op, arg1, arg2})
	return len(sa.triads)
}

func (sa *SemanticAnalyzer) generateExpr(expr ExprNode) (operand string, typ string, err error) {
	switch e := expr.(type) {
	case *IntConstNode:
		return e.Value, "int", nil

	case *IdentifierNode:
		// строковая константа в кавычках
		if len(e.Name) >= 2 && e.Name[0] == '"' && e.Name[len(e.Name)-1] == '"' {
			return e.Name, "string", nil
		}
		sym, ok := sa.curTable.Lookup(e.Name)
		if !ok {
			return "", "", fmt.Errorf("необъявленная переменная '%s'", e.Name)
		}
		if !sym.Initialized && !sym.IsArray && !sym.IsFunction {
			return "", "", fmt.Errorf("использование неинициализированной переменной '%s'", e.Name)
		}
		return e.Name, sym.Type, nil

	case *BinaryExprNode:
		// оператор вывода <<
		if e.Op == "<<" {
			leftOp, leftTyp, err := sa.generateExpr(e.Left)
			if err != nil {
				return "", "", err
			}
			rightOp, rightTyp, err := sa.generateExpr(e.Right)
			if err != nil {
				return "", "", err
			}
			if leftTyp != "ostream" {
				return "", "", fmt.Errorf("левый операнд '<<' должен быть потоком (ostream)")
			}
			if rightTyp != "int" && rightTyp != "string" && rightTyp != "ostream" {
				return "", "", fmt.Errorf("оператор << не поддерживает тип %s", rightTyp)
			}
			triNum := sa.addTriad("<<", leftOp, rightOp)
			return fmt.Sprintf("^%d", triNum), "ostream", nil
		}
		// арифметика
		if e.Op == "+" || e.Op == "-" || e.Op == "*" || e.Op == "/" {
			leftOp, leftTyp, err := sa.generateExpr(e.Left)
			if err != nil {
				return "", "", err
			}
			rightOp, rightTyp, err := sa.generateExpr(e.Right)
			if err != nil {
				return "", "", err
			}
			if leftTyp != "int" || rightTyp != "int" {
				return "", "", fmt.Errorf("операция '%s' требует целочисленных операндов", e.Op)
			}
			triNum := sa.addTriad(e.Op, leftOp, rightOp)
			return fmt.Sprintf("^%d", triNum), "int", nil
		}
		// сравнения
		if e.Op == "<" || e.Op == ">" || e.Op == "<=" || e.Op == ">=" || e.Op == "==" || e.Op == "!=" {
			leftOp, leftTyp, err := sa.generateExpr(e.Left)
			if err != nil {
				return "", "", err
			}
			rightOp, rightTyp, err := sa.generateExpr(e.Right)
			if err != nil {
				return "", "", err
			}
			if leftTyp != "int" || rightTyp != "int" {
				return "", "", fmt.Errorf("сравнение требует целочисленных операндов")
			}
			triNum := sa.addTriad(e.Op, leftOp, rightOp)
			return fmt.Sprintf("^%d", triNum), "int", nil
		}
		return "", "", fmt.Errorf("неподдерживаемый бинарный оператор '%s'", e.Op)

	case *UnaryExprNode:
		if e.Op == "++" || e.Op == "--" {
			rhsOp, rhsTyp, err := sa.generateExpr(e.Right)
			if err != nil {
				return "", "", err
			}
			if rhsTyp != "int" {
				return "", "", fmt.Errorf("операция '%s' применима только к целым", e.Op)
			}
			op := e.Op[:1] // "+" или "-"
			oneTriad := sa.addTriad(op, rhsOp, "1")
			sa.addTriad(":=", rhsOp, fmt.Sprintf("^%d", oneTriad))
			return rhsOp, rhsTyp, nil
		}
		return "", "", fmt.Errorf("неподдерживаемая унарная операция '%s'", e.Op)

	case *IndexExprNode:
		arrOp, _, err := sa.generateExpr(e.Array)
		if err != nil {
			return "", "", err
		}
		sym, ok := sa.curTable.Lookup(arrOp)
		if !ok || !sym.IsArray {
			return "", "", fmt.Errorf("индексация возможна только для массивов")
		}
		idxOp, idxTyp, err := sa.generateExpr(e.Index)
		if err != nil {
			return "", "", err
		}
		if idxTyp != "int" {
			return "", "", fmt.Errorf("индекс должен быть целым")
		}
		triNum := sa.addTriad("[]", arrOp, idxOp)
		elemType := "int"
		if sym.Type == "int[]" || sym.Type == "int[10]" {
			elemType = "int"
		}
		return fmt.Sprintf("^%d", triNum), elemType, nil

	case *CallExprNode:
		funcId, ok := e.Callee.(*IdentifierNode)
		if !ok {
			return "", "", fmt.Errorf("вызов возможен только по имени функции")
		}
		sym, ok := sa.curTable.Lookup(funcId.Name)
		if !ok {
			return "", "", fmt.Errorf("функция '%s' не объявлена", funcId.Name)
		}
		var args []string
		for _, arg := range e.Args {
			op, _, err := sa.generateExpr(arg)
			if err != nil {
				return "", "", err
			}
			args = append(args, op)
		}
		argStr := ""
		for i, a := range args {
			if i > 0 {
				argStr += ","
			}
			argStr += a
		}
		triNum := sa.addTriad("call", funcId.Name, argStr)
		retType := sym.Type
		if retType == "void" {
			return "", "void", nil
		}
		return fmt.Sprintf("^%d", triNum), retType, nil

	case *ArrayInitNode:
		return "", "", fmt.Errorf("список инициализации не может быть использован в выражении")

	default:
		return "", "", fmt.Errorf("неподдерживаемый тип выражения: %T", expr)
	}
}

// ---------- Обход AST ----------
func (sa *SemanticAnalyzer) VisitProgram(prog *ProgramNode) {
	// Первый проход: регистрируем функции
	for _, fn := range prog.Functions {
		sa.globalTable.DefineFunction(fn.Name, fn.ReturnType)
	}
	// Второй проход: анализируем тела
	for _, fn := range prog.Functions {
		sa.VisitFunction(fn)
	}
}

func (sa *SemanticAnalyzer) VisitFunction(fn *FunctionDefNode) {
	prevTable := sa.curTable
	sa.curTable = NewSymbolTable(fn.Name, prevTable)
	sa.allTables = append(sa.allTables, sa.curTable)
	defer func() { sa.curTable = prevTable }()

	sa.funcRetType = fn.ReturnType

	// Параметры
	for _, p := range fn.Params {
		typ := p.Type
		if p.IsArray {
			typ = typ + "[]"
		}
		if err := sa.curTable.Define(p.Name, typ, 0, true, p.IsArray, 0); err != nil {
			sa.errors = append(sa.errors, err.Error())
		}
	}

	sa.VisitCompoundStmt(fn.Body)

	if sa.funcRetType != "void" && fn.Name == "main" && sa.funcRetType == "int" {
		sa.addTriad("return", "0", "")
	}
}

func (sa *SemanticAnalyzer) VisitCompoundStmt(stmt *CompoundStmtNode) {
	prevTable := sa.curTable
	sa.curTable = NewSymbolTable("block", prevTable)
	sa.allTables = append(sa.allTables, sa.curTable)
	defer func() { sa.curTable = prevTable }()

	for _, s := range stmt.Statements {
		sa.VisitStatement(s)
	}
}

func (sa *SemanticAnalyzer) VisitStatement(stmt StmtNode) {
	switch s := stmt.(type) {
	case *VarDeclNode:
		typ := s.Type
		isArray := s.IsArray
		arraySize := 0
		if s.ArraySize != nil {
			if sizeConst, ok := s.ArraySize.(*IntConstNode); ok {
				fmt.Sscanf(sizeConst.Value, "%d", &arraySize)
			}
		}
		init := false

		if s.Init != nil {
			if arrInit, ok := s.Init.(*ArrayInitNode); ok {
				init = true
				for i, valExpr := range arrInit.Values {
					valOp, valTyp, err := sa.generateExpr(valExpr)
					if err != nil {
						sa.errors = append(sa.errors, err.Error())
						continue
					}
					if valTyp != typ {
						sa.errors = append(sa.errors, fmt.Sprintf("тип элемента %d не соответствует объявленному", i))
					}
					elemAddr := sa.addTriad("[]", s.Name, fmt.Sprintf("%d", i))
					sa.addTriad(":=", fmt.Sprintf("^%d", elemAddr), valOp)
				}
			} else {
				operand, typExpr, err := sa.generateExpr(s.Init)
				if err != nil {
					sa.errors = append(sa.errors, err.Error())
					return
				}
				if typExpr != typ {
					sa.errors = append(sa.errors, fmt.Sprintf("несоответствие типов: '%s' (%s) и инициализатор (%s)", s.Name, typ, typExpr))
					return
				}
				sa.addTriad(":=", s.Name, operand)
				init = true
			}
		}

		if err := sa.curTable.Define(s.Name, typ, 0, init, isArray, arraySize); err != nil {
			sa.errors = append(sa.errors, err.Error())
		}

	case *AssignStmtNode:
		var leftName string
		var leftType string
		var leftSym *Symbol

		switch l := s.Left.(type) {
		case *IdentifierNode:
			sym, ok := sa.curTable.Lookup(l.Name)
			if !ok {
				sa.errors = append(sa.errors, fmt.Sprintf("необъявленная переменная '%s'", l.Name))
				return
			}
			leftName = l.Name
			leftType = sym.Type
			leftSym = sym
		case *IndexExprNode:
			op, typ, err := sa.generateExpr(l)
			if err != nil {
				sa.errors = append(sa.errors, err.Error())
				return
			}
			leftName = op
			leftType = typ
		default:
			sa.errors = append(sa.errors, "недопустимая левая часть присваивания")
			return
		}

		rightOp, rightType, err := sa.generateExpr(s.Right)
		if err != nil {
			sa.errors = append(sa.errors, err.Error())
			return
		}

		if leftType != rightType {
			sa.errors = append(sa.errors, fmt.Sprintf("несоответствие типов: левая часть (%s), правая (%s)", leftType, rightType))
			return
		}

		sa.addTriad(":=", leftName, rightOp)
		if leftSym != nil {
			leftSym.Initialized = true
		}

	case *IfStmtNode:
		condOp, condType, err := sa.generateExpr(s.Condition)
		if err != nil {
			sa.errors = append(sa.errors, err.Error())
			return
		}
		if condType != "int" {
			sa.errors = append(sa.errors, "условие должно быть целочисленным")
		}
		sa.addTriad("if", condOp, "")
		sa.VisitCompoundStmt(s.Then)
		if s.Else != nil {
			sa.addTriad("else", "", "")
			sa.VisitCompoundStmt(s.Else)
		}
		sa.addTriad("endif", "", "")

	case *ForStmtNode:
		if s.Init != nil {
			sa.VisitStatement(s.Init)
		}
		loopStart := len(sa.triads) + 1
		if s.Condition != nil {
			condOp, condType, err := sa.generateExpr(s.Condition)
			if err != nil {
				sa.errors = append(sa.errors, err.Error())
				return
			}
			if condType != "int" {
				sa.errors = append(sa.errors, "условие цикла должно быть целочисленным")
			}
			sa.addTriad("while", condOp, "")
		} else {
			sa.addTriad("while", "1", "")
		}
		sa.VisitCompoundStmt(s.Body)
		if s.Post != nil {
			_, _, err := sa.generateExpr(s.Post)
			if err != nil {
				sa.errors = append(sa.errors, err.Error())
			}
		}
		sa.addTriad("goto", fmt.Sprintf("^%d", loopStart), "")
		sa.addTriad("endwhile", "", "")

	case *ReturnStmtNode:
		var operand string
		var typ string
		var err error
		if s.Value != nil {
			operand, typ, err = sa.generateExpr(s.Value)
			if err != nil {
				sa.errors = append(sa.errors, err.Error())
				return
			}
			if typ != sa.funcRetType {
				sa.errors = append(sa.errors, fmt.Sprintf("тип возврата: ожидается %s, получен %s", sa.funcRetType, typ))
				return
			}
		} else {
			if sa.funcRetType != "void" {
				sa.errors = append(sa.errors, fmt.Sprintf("функция должна вернуть %s", sa.funcRetType))
				return
			}
			operand = ""
		}
		sa.addTriad("return", operand, "")

	case *ExprStmtNode:
		_, _, err := sa.generateExpr(s.Expr)
		if err != nil {
			sa.errors = append(sa.errors, err.Error())
		}

	default:
		sa.errors = append(sa.errors, fmt.Sprintf("неподдерживаемый оператор: %T", stmt))
	}
}

// PrintSymbolTable выводит все символы из всех областей видимости
func (sa *SemanticAnalyzer) PrintSymbolTable() {
	fmt.Println("Таблица символов:")
	fmt.Println("Имя     | Тип           | Область видимости | Инициализирована")
	fmt.Println("--------+---------------+-------------------+-----------------")

	// Проходим по всем сохранённым таблицам
	for _, st := range sa.allTables {
		for _, sym := range st.symbols {
			initStr := "нет"
			if sym.Initialized {
				initStr = "да"
			}
			fmt.Printf("%-8s| %-13s | %-17s | %s\n", sym.Name, sym.Type, sym.Scope, initStr)
		}
	}
}

func (sa *SemanticAnalyzer) PrintTriads() {
	fmt.Println("\nПромежуточное представление (триады):")
	for i, t := range sa.triads {
		fmt.Printf("%d) %s\n", i+1, t.String())
	}
}

func (sa *SemanticAnalyzer) Analyze(prog *ProgramNode) bool {
	sa.VisitProgram(prog)
	if len(sa.errors) > 0 {
		fmt.Println("Семантические ошибки:")
		for _, err := range sa.errors {
			fmt.Println("  -", err)
		}
		return false
	}
	sa.PrintSymbolTable()
	sa.PrintTriads()
	return true
}
