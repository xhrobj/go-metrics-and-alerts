// Package noosexit содержит анализатор, запрещающий прямой вызов os.Exit внутри функции main пакета main.
package noosexit

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const diagnostic = "os.Exit call in main function is prohibited"

// Analyzer проверяет прямые вызовы os.Exit внутри функции main пакета main.
var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "prohibits direct os.Exit calls in main function of main package",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" || strings.HasSuffix(pass.Pkg.Path(), ".test") {
		return nil, nil
	}

	for _, file := range pass.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" || fn.Body == nil {
				continue
			}

			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}

				if isOSExitCall(pass.TypesInfo, call) {
					pass.Reportf(call.Pos(), diagnostic)
				}

				return true
			})
		}
	}

	return nil, nil
}

func isOSExitCall(info *types.Info, call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Exit" {
		return false
	}

	pkgIdent, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	pkgName, ok := info.Uses[pkgIdent].(*types.PkgName)
	if !ok {
		return false
	}

	return pkgName.Imported().Path() == "os"
}
