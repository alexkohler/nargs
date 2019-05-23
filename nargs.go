package nargs

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"log"

	"golang.org/x/tools/go/packages"
)

func init() {
	build.Default.UseAllFiles = true
}

//TODO - should look for exprstmt then check the returns?

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
	f                   *token.FileSet
	pkg                 *packages.Package
	results             []string
	includeNamedReturns bool
	includeReceivers    bool
	errsFound           bool
}

// CheckForUnusedFunctionArgs will parse the files/packages contained in args
// and walk the AST searching for unused function parameters.
func CheckForUnusedFunctionArgs(args []string, flags Flags) (results []string, exitWithStatus bool, _ error) {
	// We'll probably only want to accept packges.

	//TODO wire in slice for go list -f '{{ .Imports }}' github.com/alexkohler/nargs
	pkgsStr := []string{"github.com/alexkohler/nargs/testdata", "fmt", "go/ast", "go/build", "go/parser", "go/token", "go/types", "golang.org/x/tools/go/packages", "log", "os", "path", "path/filepath", "regexp", "runtime", "strings"}

	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: false,
		// BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(c.Tags, " "))},
	}

	pkgs, err := packages.Load(cfg, pkgsStr...)
	if err != nil {
		panic(err)
	}

	retVis := &unusedVisitor{
		includeNamedReturns: flags.IncludeNamedReturns,
		includeReceivers:    flags.IncludeReceivers,
	}

	// var wg sync.WaitGroup
	for _, pkg := range pkgs {
		if pkg.Name == "main" {
			// go func(pkg *packages.Package) {
			// defer wg.Done()
			log.Printf("Checking %s\n", pkg.Types.Path())
			retVis.pkg = pkg
			for _, astFile := range pkg.Syntax {
				fmt.Printf("looking at file %v\n", astFile.Name.Name)
				ast.Walk(retVis, astFile)
			}
			// }(pkg)
		}
	}

	return retVis.results, retVis.errsFound && flags.SetExitStatus, nil
}

func (v *unusedVisitor) hasVoidReturn(call *ast.CallExpr) (ret bool) {
	if call == nil {
		return true
	}
	if v.pkg == nil || v.pkg.TypesInfo == nil {
		return true
	}
	if _, ok := v.pkg.TypesInfo.Types[call]; !ok {
		return true
	}
	defer func() { fmt.Printf("checking out %+v: %v\n", call.Fun, ret) }()
	switch t := v.pkg.TypesInfo.Types[call].Type.(type) {
	case *types.Named:
		// fmt.Printf("sangle dangle\n")
	case *types.Pointer:
		// fmt.Printf("sangle dangle 2\n")
	case *types.Tuple:
		fmt.Printf("length %v\n", t.Len())
		return t.Len() == 0
	case *types.Slice:
	default:
		// fmt.Printf("defaa %Ta\n", t)
	}

	return false
	// switch t := v.pkg.TypesInfo.Types[call].Type.(type) {
	// case *types.Named:
	// 	// Single return
	// 	return []bool{isErrorType(t)}
	// case *types.Pointer:
	// 	// Single return via pointer
	// 	return []bool{isErrorType(t)}
	// case *types.Tuple:
	// 	// Multiple returns
	// 	s := make([]bool, t.Len())
	// 	for i := 0; i < t.Len(); i++ {
	// 		switch et := t.At(i).Type().(type) {
	// 		case *types.Named:
	// 			// Single return
	// 			s[i] = isErrorType(et)
	// 		case *types.Pointer:
	// 			// Single return via pointer
	// 			s[i] = isErrorType(et)
	// 		default:
	// 			s[i] = false
	// 		}
	// 	}
	// 	return s
	// }
	// return []bool{false}
}

// Visit implements the ast.Visitor Visit method.
func (v *unusedVisitor) Visit(node ast.Node) ast.Visitor {
	// search for call expressions
	funcDecl, ok := node.(*ast.FuncDecl)
	if !ok {
		return v
	}

	paramMap := make(map[string]bool)

	// We cannot exit if len(paramMap) == 0, we may have a function closure with
	// unused variables

	// file := v.f.File(funcDecl.Pos())

	// Analyze body of function
	for funcDecl.Body != nil && len(funcDecl.Body.List) != 0 {
		fmt.Printf("exploring %v\n", funcDecl.Name.Name)
		stmt := funcDecl.Body.List[0]
		switch s := stmt.(type) {
		case *ast.IfStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Init, s.Body, s.Else)
			funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{s.Cond}, funcDecl.Body.List)

		case *ast.AssignStmt:
			//TODO see if variables on LHS are used? i.e. add them to param map?
			funcDecl.Body.List = v.handleExprs(paramMap, s.Lhs, funcDecl.Body.List)
			funcDecl.Body.List = v.handleExprs(paramMap, s.Rhs, funcDecl.Body.List)

		case *ast.BlockStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.List...)

		case *ast.ReturnStmt:
			funcDecl.Body.List = v.handleExprs(paramMap, s.Results, funcDecl.Body.List)

		case *ast.DeclStmt:
			switch d := s.Decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch specType := spec.(type) {
					case *ast.ValueSpec:
						handleIdents(paramMap, specType.Names)
						funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{specType.Type}, funcDecl.Body.List)
						funcDecl.Body.List = v.handleExprs(paramMap, specType.Values, funcDecl.Body.List)

					case *ast.TypeSpec:
						handleIdent(paramMap, specType.Name)
						funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{specType.Type}, funcDecl.Body.List)

					default:
						log.Printf("ERROR: unknown spec type %T\n", specType)
					}
				}

			default:
				log.Printf("ERROR: unknown decl type %T\n", d)
			}

		case *ast.ExprStmt:
			callExpr, ok := s.X.(*ast.CallExpr)
			if ok {
				switch c := callExpr.Fun.(type) {
				case *ast.Ident:
					// fmt.Print("wat's going on?\n")
					if !v.hasVoidReturn(callExpr) {
						fmt.Printf("pin it bud %v\n", c.Name)
					}
				case *ast.SelectorExpr:
					x, ok := c.X.(*ast.Ident)
					if !ok {
						fmt.Println("kate mccannon")
						continue
					}
					if !v.hasVoidReturn(callExpr) {
						fmt.Printf("pin it bud %v.%v\n", x.Name, c.Sel.Name)
					}
				}

			}

			// funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{s.X}, funcDecl.Body.List)

		case *ast.RangeStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body)
			funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{s.X}, funcDecl.Body.List)

		case *ast.ForStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body)
			funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{s.Cond}, funcDecl.Body.List)

			funcDecl.Body.List = append(funcDecl.Body.List, s.Post)

		case *ast.TypeSwitchStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body, s.Assign, s.Init)

		case *ast.CaseClause:
			funcDecl.Body.List = v.handleExprs(paramMap, s.List, funcDecl.Body.List)

			funcDecl.Body.List = append(funcDecl.Body.List, s.Body...)

		case *ast.SendStmt:
			funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{s.Chan, s.Value}, funcDecl.Body.List)

		case *ast.GoStmt:
			funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{s.Call}, funcDecl.Body.List)

		case *ast.DeferStmt:
			funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{s.Call}, funcDecl.Body.List)

		case *ast.SelectStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body)

		case *ast.CommClause:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body...)
			funcDecl.Body.List = append(funcDecl.Body.List, s.Comm)

		case *ast.BranchStmt:
			handleIdent(paramMap, s.Label)

		case *ast.SwitchStmt:
			funcDecl.Body.List = append(funcDecl.Body.List, s.Body, s.Init)
			funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{s.Tag}, funcDecl.Body.List)

		case *ast.LabeledStmt:
			handleIdent(paramMap, s.Label)
			funcDecl.Body.List = append(funcDecl.Body.List, s.Stmt)

		case *ast.IncDecStmt:
			funcDecl.Body.List = v.handleExprs(paramMap, []ast.Expr{s.X}, funcDecl.Body.List)

		case nil, *ast.EmptyStmt:
			//no-op

		default:
			log.Printf("ERROR: unknown stmt type %T\n", s)
		}

		funcDecl.Body.List = funcDecl.Body.List[1:]
	}

	// for funcName, used := range paramMap {
	// 	if !used {
	// 		if file != nil {
	// 			if funcDecl.Name != nil {
	// 				//TODO print parameter vs parameter(s)?
	// 				//TODO differentiation of used parameter vs. receiver?
	// 				v.results = append(v.results, fmt.Sprintf("%v:%v %v contains unused parameter %v\n", file.Name(), file.Position(funcDecl.Pos()).Line, funcDecl.Name.Name, funcName))
	// 				v.errsFound = true
	// 			}
	// 		}
	// 	}
	// }

	return v
}

func nilFunc() int { return 1 }
func handleIdents(paramMap map[string]bool, identList []*ast.Ident) {
	nilFunc()
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
		} else {
			paramMap[ident.Obj.Name] = false
		}
	}

	//TODO - ensure this truly isn't needed - can we rely on the
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
			exprList = append(exprList, e.X) //TODO, do we need to then worry about x.left being used?
			exprList = append(exprList, e.Y) //TODO, do we need to then worry about x.left being used?

		case *ast.CallExpr:
			// fmt.Println("got some call exprs :))))")
			// v.errorsByArg(e)
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
			// nothing to do here, this is a type (i.e. name will be "int")

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
			exprList, stmtList = handleFieldList(paramMap, e.Fields, exprList, stmtList)

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
