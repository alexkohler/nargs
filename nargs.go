package nargs

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"log"
)

func init() {
	build.Default.UseAllFiles = true
}

type unusedVisitor struct {
	f *token.FileSet
}

// CheckForUnusedFunctionArgs will parse the files/packages contained in args
// and walk the AST searching for unused function parameters.
func CheckForUnusedFunctionArgs(args []string) error {

	fset := token.NewFileSet()

	files, err := parseInput(args, fset)
	if err != nil {
		return fmt.Errorf("could not parse input %v", err)
	}

	retVis := &unusedVisitor{
		f: fset,
	}

	for _, f := range files {
		ast.Walk(retVis, f)
	}

	return nil
}

// Visit implements the ast.Visitor Visit method.
func (v *unusedVisitor) Visit(node ast.Node) ast.Visitor {

	// search for call expressions
	funcDecl, ok := node.(*ast.FuncDecl)
	if !ok {
		return v
	}

	paramMap := make(map[string]bool)

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

		if funcDecl.Type.Results != nil {
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

	if funcDecl.Recv != nil {
		for _, field := range funcDecl.Recv.List {
			for _, name := range field.Names {
				if name.Name == "_" {
					continue
				}
				paramMap[name.Name] = false
			}
		}
	}

	if len(paramMap) == 0 {
		return v
	}

	file := v.f.File(funcDecl.Pos())

	// Analyze body of function
	for funcDecl.Body != nil && len(funcDecl.Body.List) != 0 {
		stmt := funcDecl.Body.List[0]

		switch s := stmt.(type) {
		case *ast.IfStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Init, s.Body, s.Else)
			funcDecl.Body.List = handleExprs(paramMap, []ast.Expr{s.Cond}, funcDecl.Body.List)

		case *ast.AssignStmt:
			//TODO check both left and right sides?
			funcDecl.Body.List = handleExprs(paramMap, s.Lhs, funcDecl.Body.List)
			funcDecl.Body.List = handleExprs(paramMap, s.Rhs, funcDecl.Body.List)

		case *ast.BlockStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.List...)

		case *ast.ReturnStmt:
			funcDecl.Body.List = handleExprs(paramMap, s.Results, funcDecl.Body.List)

		case *ast.DeclStmt:
			switch d := s.Decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch specType := spec.(type) {
					case *ast.ValueSpec:
						handleIdents(paramMap, specType.Names)
						funcDecl.Body.List = handleExprs(paramMap, []ast.Expr{specType.Type}, funcDecl.Body.List)
						funcDecl.Body.List = handleExprs(paramMap, specType.Values, funcDecl.Body.List)

					case *ast.TypeSpec:
						handleIdent(paramMap, specType.Name)
						funcDecl.Body.List = handleExprs(paramMap, []ast.Expr{specType.Type}, funcDecl.Body.List)

					default:
						log.Printf("ERROR: unknown spec type %T\n", specType)
					}
				}

			default:
				log.Printf("ERROR: unknown decl type %T\n", d)
			}

		case *ast.ExprStmt:
			funcDecl.Body.List = handleExprs(paramMap, []ast.Expr{s.X}, funcDecl.Body.List)

		case *ast.RangeStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body)
			funcDecl.Body.List = handleExprs(paramMap, []ast.Expr{s.X}, funcDecl.Body.List)

		case *ast.ForStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body)
			funcDecl.Body.List = handleExprs(paramMap, []ast.Expr{s.Cond}, funcDecl.Body.List)

			funcDecl.Body.List = append(funcDecl.Body.List, s.Post)

		case *ast.TypeSwitchStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body, s.Assign, s.Init)

		case *ast.CaseClause:
			funcDecl.Body.List = handleExprs(paramMap, s.List, funcDecl.Body.List)

			funcDecl.Body.List = append(funcDecl.Body.List, s.Body...)

		case *ast.SendStmt:
			funcDecl.Body.List = handleExprs(paramMap, []ast.Expr{s.Chan, s.Value}, funcDecl.Body.List)

		case *ast.GoStmt:
			funcDecl.Body.List = handleExprs(paramMap, []ast.Expr{s.Call}, funcDecl.Body.List)

		case *ast.DeferStmt:
			funcDecl.Body.List = handleExprs(paramMap, []ast.Expr{s.Call}, funcDecl.Body.List)

		case *ast.SelectStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body)

		case *ast.CommClause:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body...)
			funcDecl.Body.List = append(funcDecl.Body.List, s.Comm)

		case *ast.BranchStmt:
			handleIdent(paramMap, s.Label)

		case *ast.SwitchStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body, s.Init)
			funcDecl.Body.List = handleExprs(paramMap, []ast.Expr{s.Tag}, funcDecl.Body.List)

		case *ast.LabeledStmt:
			// this one is kinda weird
			handleIdent(paramMap, s.Label)
			funcDecl.Body.List = append(funcDecl.Body.List, s.Stmt)

		case nil, *ast.IncDecStmt, *ast.EmptyStmt:
			//no-op

		default:
			// nils will happen here without nil checks on my appends, meh
			log.Printf("ERROR: unknown stmt type %T\n", s)

		}

		funcDecl.Body.List = funcDecl.Body.List[1:]
	}

	for funcName, used := range paramMap {
		if !used {
			if file != nil {
				if funcDecl.Name != nil {
					//TODO print parameter vs parameter(s)?
					log.Printf("%v:%v %v contains unused parameter %v\n", file.Name(), file.Position(funcDecl.Pos()).Line, funcDecl.Name.Name, funcName)
				}
			}
		}
	}

	return v
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
		paramMap[ident.Obj.Name] = true
	}

	if _, ok := paramMap[ident.Name]; ok {
		paramMap[ident.Name] = true
	}
}

func handleExprs(paramMap map[string]bool, exprList []ast.Expr, stmtList []ast.Stmt) []ast.Stmt {
	for len(exprList) != 0 {
		expr := exprList[0]
		switch e := expr.(type) {
		case *ast.Ident:
			handleIdent(paramMap, e)

		case *ast.BinaryExpr:
			exprList = append(exprList, e.X) //TODO, do we need to then worry about x.left being used?
			exprList = append(exprList, e.Y) //TODO, do we need to then worry about x.left being used?

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
			exprList = append(exprList, e.X)

		case *ast.TypeAssertExpr:
			exprList = append(exprList, e.X, e.Type)

		case *ast.UnaryExpr:
			exprList = append(exprList, e.X)

		case *ast.BasicLit:
			// nothing to do here, no variable name

		case *ast.FuncLit:
			stmtList = append(stmtList, e.Body)

		case *ast.CompositeLit:
			exprList = append(exprList, e.Elts...)

		case *ast.ArrayType:
			exprList = append(exprList, e.Elt, e.Len)

		case *ast.ChanType:
			exprList = append(exprList, e.Value)

		case *ast.FuncType:
			exprList, stmtList = processFieldList(paramMap, e.Params, exprList, stmtList)
			exprList, stmtList = processFieldList(paramMap, e.Results, exprList, stmtList)

		case *ast.InterfaceType:
			exprList, stmtList = processFieldList(paramMap, e.Methods, exprList, stmtList)

		case *ast.MapType:
			exprList = append(exprList, e.Key, e.Value)

		case *ast.StructType:
			exprList, stmtList = processFieldList(paramMap, e.Fields, exprList, stmtList)

		case *ast.Ellipsis:
			exprList = append(exprList, e.Elt)

		case nil:
			// no op

		default:
			log.Printf("ERROR: unknown expr type %T\n", e)
		}
		exprList = exprList[1:]
	}

	return stmtList
}

func processFieldList(paramMap map[string]bool, fieldList *ast.FieldList, exprList []ast.Expr, stmtList []ast.Stmt) ([]ast.Expr, []ast.Stmt) {
	if fieldList == nil {
		return exprList, stmtList
	}

	for _, field := range fieldList.List {
		exprList = append(exprList, field.Type)
		handleIdents(paramMap, field.Names)

		// don't care about Tag, need to handle ident and expr
	}
	return exprList, stmtList
}
