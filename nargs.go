package nargs

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"log"
	"sort"
	"strconv"
	"strings"
)

func init() {
	build.Default.UseAllFiles = true
}

// Flags contains configuration specific to nargs
// * IncludeTests - include test files in analysis
// * SetExitStatus - set exit status to 1 if any issues are found
// * IncludeNamedReturns - include unused named returns
// * IncludeReceivers - include unused receivers
type Flags struct {
	IncludeTests        bool
	SetExitStatus       bool
	IncludeNamedReturns bool
	IncludeReceivers    bool
}

type unusedVisitor struct {
	fileSet             *token.FileSet
	resultsSet          map[string]struct{}
	includeNamedReturns bool
	includeReceivers    bool
	errsFound           bool
}

// CheckForUnusedFunctionArgs will parse the files/packages contained in args
// and walk the AST searching for unused function parameters.
func CheckForUnusedFunctionArgs(args []string, flags Flags) (results []string, exitWithStatus bool, _ error) {
	fset := token.NewFileSet()
	files, err := parseInput(args, fset, flags.IncludeTests)
	if err != nil {
		return nil, false, fmt.Errorf("could not parse input, %v", err)
	}

	retVis := &unusedVisitor{
		fileSet:             fset,
		includeNamedReturns: flags.IncludeNamedReturns,
		includeReceivers:    flags.IncludeReceivers,
		resultsSet:          make(map[string]struct{}),
	}

	// visitorResult contains the results for a specific visitor and is cleared on each
	// iteration
	var visitorResult []string
	for _, f := range files {
		if f == nil {
			continue
		}
		ast.Walk(retVis, f)
		for result := range retVis.resultsSet {
			visitorResult = append(visitorResult, result)
		}
		// Due to our analysis, of the ast.File, we may end up getting our results out of order. Sort by line number to keep
		// the results in a consistent format.
		sort.Sort(byLineNumber(visitorResult))
		results = append(results, visitorResult...)
		visitorResult = nil
		retVis.resultsSet = make(map[string]struct{})
	}

	return results, retVis.errsFound && flags.SetExitStatus, nil
}

// ugly, but not sure if there's another option..
type byLineNumber []string

func (a byLineNumber) Len() int { return len(a) }
func (a byLineNumber) Less(i, j int) bool {
	iLine := strings.Split(a[i], ":")
	num := strings.Split(iLine[1], " ")
	iNum, _ := strconv.Atoi(num[0])

	jLine := strings.Split(a[j], ":")
	num = strings.Split(jLine[1], " ")
	jNum, _ := strconv.Atoi(num[0])
	return iNum < jNum
}
func (a byLineNumber) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Visit implements the ast.Visitor Visit method.
func (v *unusedVisitor) Visit(node ast.Node) ast.Visitor {
	var stmtList []ast.Stmt
	var file *token.File
	paramMap := make(map[string]bool)
	var funcDecl *ast.FuncDecl

	switch topLevelType := node.(type) {
	case *ast.FuncDecl:
		funcDecl = topLevelType
		if funcDecl.Body == nil {
			// This means funcDecl is an external (non-Go) function, these
			// should not be included in the analysis
			return v
		}
		stmtList = v.handleFuncDecl(paramMap, funcDecl, stmtList)
		file = v.fileSet.File(funcDecl.Pos())

	case *ast.File:
		file = v.fileSet.File(topLevelType.Pos())
		if topLevelType.Decls != nil {
			stmtList = v.handleDecls(paramMap, topLevelType.Decls, stmtList)
		}

	default:
		return v

	}

	// We cannot exit if len(paramMap) == 0, we may have a function closure with
	// unused variables

	// Analyze body of function
	v.handleStmts(paramMap, stmtList)

	for paramName, used := range paramMap {
		if used {
			continue
		}
		if file == nil {
			continue
		}
		if funcDecl == nil {
			continue
		}
		if funcDecl.Name == nil {
			continue
		}

		// TODO print parameter vs parameter(s)?
		// TODO differentiation of used parameter vs. receiver?
		resStr := fmt.Sprintf("%v:%v %v contains unused parameter %v\n", file.Name(), file.Position(funcDecl.Pos()).Line, funcDecl.Name.Name, paramName)
		v.resultsSet[resStr] = struct{}{}
		v.errsFound = true
	}

	return v
}

func (v *unusedVisitor) handleStmts(paramMap map[string]bool, stmtList []ast.Stmt) {
	for len(stmtList) != 0 {
		stmt := stmtList[0]
		switch s := stmt.(type) {
		case *ast.IfStmt:
			stmtList = append(stmtList, s.Init, s.Body, s.Else)
			stmtList = v.handleExprs(paramMap, []ast.Expr{s.Cond}, stmtList)

		case *ast.AssignStmt:
			assigned := false
			for index, right := range s.Rhs {
				funcLit, ok := right.(*ast.FuncLit)
				if !ok {
					continue
				}
				funcName, ok := s.Lhs[index].(*ast.Ident)
				if !ok {
					// TODO - understand this case a little more
					// log.Printf("@@@@@@@@@@@@@@@@@@@@@@@@@2 wat")
					continue
				}
				v.handleFuncLit(paramMap, funcLit, funcName)
				assigned = true
			}

			if !assigned {
				stmtList = v.handleExprs(paramMap, s.Lhs, stmtList)
				stmtList = v.handleExprs(paramMap, s.Rhs, stmtList)
			}

		case *ast.BlockStmt:
			stmtList = append(stmtList, s.List...)

		case *ast.ReturnStmt:
			stmtList = v.handleExprs(paramMap, s.Results, stmtList)

		case *ast.DeclStmt:
			stmtList = v.handleDecls(paramMap, []ast.Decl{s.Decl}, stmtList)

		case *ast.ExprStmt:
			stmtList = v.handleExprs(paramMap, []ast.Expr{s.X}, stmtList)

		case *ast.RangeStmt:
			stmtList = append(stmtList, s.Body)
			stmtList = v.handleExprs(paramMap, []ast.Expr{s.X}, stmtList)

		case *ast.ForStmt:
			stmtList = append(stmtList, s.Init)

			stmtList = append(stmtList, s.Body)
			stmtList = v.handleExprs(paramMap, []ast.Expr{s.Cond}, stmtList)

			stmtList = append(stmtList, s.Post)

		case *ast.TypeSwitchStmt:
			stmtList = append(stmtList, s.Body, s.Assign, s.Init)

		case *ast.CaseClause:
			stmtList = v.handleExprs(paramMap, s.List, stmtList)

			stmtList = append(stmtList, s.Body...)

		case *ast.SendStmt:
			stmtList = v.handleExprs(paramMap, []ast.Expr{s.Chan, s.Value}, stmtList)

		case *ast.GoStmt:
			stmtList = v.handleExprs(paramMap, []ast.Expr{s.Call}, stmtList)

		case *ast.DeferStmt:
			stmtList = v.handleExprs(paramMap, []ast.Expr{s.Call}, stmtList)

		case *ast.SelectStmt:
			stmtList = append(stmtList, s.Body)

		case *ast.CommClause:
			stmtList = append(stmtList, s.Body...)
			stmtList = append(stmtList, s.Comm)

		case *ast.BranchStmt:
			handleIdent(paramMap, s.Label)

		case *ast.SwitchStmt:
			stmtList = append(stmtList, s.Body, s.Init)
			stmtList = v.handleExprs(paramMap, []ast.Expr{s.Tag}, stmtList)

		case *ast.LabeledStmt:
			handleIdent(paramMap, s.Label)
			stmtList = append(stmtList, s.Stmt)

		case *ast.IncDecStmt:
			stmtList = v.handleExprs(paramMap, []ast.Expr{s.X}, stmtList)

		case nil, *ast.EmptyStmt:
			// no-op

		default:
			log.Printf("ERROR: unknown stmt type %T\n", s)
		}

		stmtList = stmtList[1:]
	}
}

func handleIdents(paramMap map[string]bool, identList []*ast.Ident) {
	for _, ident := range identList {
		handleIdent(paramMap, ident)
	}
}

func handleIdent(paramMap map[string]bool, ident *ast.Ident) {
	if ident == nil {
		return
	}

	if ident.Obj != nil && ident.Obj.Kind == ast.Var {
		if _, ok := paramMap[ident.Obj.Name]; ok {
			paramMap[ident.Obj.Name] = true
		}
		/*else {
			if ident.Obj.Name != "_" {
				paramMap[ident.Obj.Name] = false
			}
		}*/
	}

	// TODO - ensure this truly isn't needed - can we rely on the
	// ident object name?
	// if _, ok := paramMap[ident.Name]; ok {
	// 	paramMap[ident.Name] = true
	// }
}

func (v *unusedVisitor) handleExprs(paramMap map[string]bool, exprList []ast.Expr, stmtList []ast.Stmt) []ast.Stmt {
	for len(exprList) != 0 {
		expr := exprList[0]
		switch e := expr.(type) {
		case *ast.Ident:
			handleIdent(paramMap, e)

		case *ast.BinaryExpr:
			exprList = append(exprList, e.X) // TODO, do we need to then worry about x.left being used?
			exprList = append(exprList, e.Y) // TODO, do we need to then worry about x.left being used?

		case *ast.CallExpr:
			exprList = append(exprList, e.Args...)
			exprList = append(exprList, e.Fun)

		case *ast.IndexExpr:
			exprList = append(exprList, e.X)
			exprList = append(exprList, e.Index)

		case *ast.KeyValueExpr:
			exprList = append(exprList, e.Key, e.Value)

		case *ast.ParenExpr:
			exprList = append(exprList, e.X)

		case *ast.SelectorExpr:
			exprList = append(exprList, e.X)
			handleIdent(paramMap, e.Sel)

		case *ast.SliceExpr:
			exprList = append(exprList, e.Low, e.High, e.Max, e.X)

		case *ast.StarExpr:
			// need to dig deeper to see if this is a derefenced type
			exprList = append(exprList, e.X)

		case *ast.TypeAssertExpr:
			exprList = append(exprList, e.X, e.Type)

		case *ast.UnaryExpr:
			exprList = append(exprList, e.X)

		case *ast.BasicLit:
			// nothing to do here, no variable name

		case *ast.FuncLit:
			exprList = append(exprList, e.Type)
			stmtList = append(stmtList, e.Body)

		case *ast.CompositeLit:
			exprList = append(exprList, e.Elts...)

		case *ast.ArrayType:
			exprList = append(exprList, e.Elt, e.Len)

		case *ast.ChanType:
			exprList = append(exprList, e.Value)

		case *ast.FuncType:
			exprList, stmtList = handleFieldList(paramMap, e.Params, exprList, stmtList)
			exprList, stmtList = handleFieldList(paramMap, e.Results, exprList, stmtList)

		case *ast.InterfaceType:
			exprList, stmtList = handleFieldList(paramMap, e.Methods, exprList, stmtList)

		case *ast.MapType:
			exprList = append(exprList, e.Key, e.Value)

		case *ast.StructType:
			// TODO: no-op,  unless it contains funcs I guess? revisit this
			// exprList, stmtList = handleFieldList(paramMap, e.Fields, exprList, stmtList)

		case *ast.Ellipsis:
			exprList = append(exprList, e.Elt)

		case *ast.IndexListExpr:
			exprList = append(exprList, e.X)
			exprList = append(exprList, e.Indices...)

		case nil:
			// no op

		default:
			log.Printf("ERROR: unknown expr type %T\n", e)
		}
		exprList = exprList[1:]
	}

	return stmtList
}

func handleFieldList(paramMap map[string]bool, fieldList *ast.FieldList, exprList []ast.Expr, stmtList []ast.Stmt) ([]ast.Expr, []ast.Stmt) {
	if fieldList == nil {
		return exprList, stmtList
	}

	for _, field := range fieldList.List {
		exprList = append(exprList, field.Type)
		handleIdents(paramMap, field.Names)
	}
	return exprList, stmtList
}

func (v *unusedVisitor) handleDecls(paramMap map[string]bool, decls []ast.Decl, initialStmts []ast.Stmt) []ast.Stmt {
	for _, decl := range decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch specType := spec.(type) {
				case *ast.ValueSpec:
					// TODO - I think the only specs we care about here are when we have a function declaration
					handleIdents(paramMap, specType.Names)
					initialStmts = v.handleExprs(paramMap, []ast.Expr{specType.Type}, initialStmts)
					initialStmts = v.handleExprs(paramMap, specType.Values, initialStmts)

					for index, value := range specType.Values {
						funcLit, ok := value.(*ast.FuncLit)
						if !ok {
							continue
						}
						funcName := specType.Names[index]
						// get arguments of function, this is a candidate
						// with potentially unused arguments
						v.handleFuncLit(paramMap, funcLit, funcName)
					}

				case *ast.TypeSpec:
					handleIdent(paramMap, specType.Name)
					initialStmts = v.handleExprs(paramMap, []ast.Expr{specType.Type}, initialStmts)

				case *ast.ImportSpec:
					// no-op, ImportSpecs do not contain functions

				default:
					log.Printf("ERROR: unknown spec type %T\n", specType)
				}
			}
		case *ast.FuncDecl:
			initialStmts = v.handleFuncDecl(paramMap, d, initialStmts)
		default:
			log.Printf("ERROR: unknown decl type %T\n", d)
		}
	}
	return initialStmts
}

// paramMap is passed in for cases where we have an outer function with a parameter
// that is captured by closure by the function literal
func (v *unusedVisitor) handleFuncLit(paramMap map[string]bool, funcLit *ast.FuncLit, funcName *ast.Ident) {
	if funcLit.Type != nil && funcLit.Type.Params != nil {
		// declare a separate parameter map for handling

		funcParamMap := make(map[string]bool)
		for _, param := range funcLit.Type.Params.List {
			for _, paramName := range param.Names {
				if paramName.Name != "_" {
					funcParamMap[paramName.Name] = false
				}
			}
		}

		// generate potential statements
		v.handleStmts(funcParamMap, []ast.Stmt{funcLit.Body})
		v.handleStmts(paramMap, []ast.Stmt{funcLit.Body})

		for paramName, used := range funcParamMap {
			if !used && paramName != "_" {
				// TODO: this append currently causes things to appear out of order (2)
				file := v.fileSet.File(funcLit.Pos())
				resStr := fmt.Sprintf("%v:%v %v contains unused parameter %v\n", file.Name(), file.Position(funcLit.Pos()).Line, funcName.Name, paramName)
				v.resultsSet[resStr] = struct{}{}
			}
		}
	}
}

func (v *unusedVisitor) handleFuncDecl(paramMap map[string]bool, funcDecl *ast.FuncDecl, initialStmts []ast.Stmt) []ast.Stmt {
	if funcDecl.Body != nil {
		initialStmts = append(initialStmts, funcDecl.Body.List...)
	}
	if funcDecl.Type != nil {
		if funcDecl.Type.Params != nil {
			for _, paramList := range funcDecl.Type.Params.List {
				for _, name := range paramList.Names {
					if name.Name == "_" {
						continue
					}
					paramMap[name.Name] = false
				}
			}
		}

		if v.includeNamedReturns && funcDecl.Type.Results != nil {
			for _, paramList := range funcDecl.Type.Results.List {
				for _, name := range paramList.Names {
					if name.Name == "_" {
						continue
					}
					paramMap[name.Name] = false
				}
			}
		}
	}

	if v.includeReceivers && funcDecl.Recv != nil {
		for _, field := range funcDecl.Recv.List {
			for _, name := range field.Names {
				if name.Name == "_" {
					continue
				}
				paramMap[name.Name] = false
			}
		}
	}

	return initialStmts
}
