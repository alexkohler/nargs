package nargs

import (
	"fmt"
	"go/ast"
)

func processExprdddd(paramMap map[string]bool, exprList []ast.Expr, stmtList []ast.Stmt) []ast.Stmt {
	for len(exprList) != 0 {
		expr := exprList[0]
		switch e := expr.(type) {
		case *ast.Ident:
			handleIdent(paramMap, e)
		case *ast.BinaryExpr:
			exprList = append(exprList, e.X) //TODO, do we need to then worry about x.left being used?
			exprList = append(exprList, e.Y) //TODO, do we need to then worry about x.left being used?
		case *ast.FuncLit:
			stmtList = append(stmtList, e.Body)
		case *ast.BasicLit:
			// nothing to do here, no variable name
		case *ast.SelectorExpr:
			exprList = append(exprList, e.X)
			handleIdent(paramMap, e.Sel)
		case *ast.CompositeLit:
			exprList = append(exprList, e.Elts...)

		case *ast.CallExpr:
			exprList = append(exprList, e.Args...)
			exprList = append(exprList, e.Fun)

		case *ast.IndexExpr:
			exprList = append(exprList, e.X)
			exprList = append(exprList, e.Index)

		default:
			fmt.Printf("@@@@@@@@@@ missing type %T\n", e)
		}
		exprList = exprList[1:]
	}

	return stmtList
}
