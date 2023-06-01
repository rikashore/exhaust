package exhaust

import (
	"go/ast"
	"go/types"
	_ "go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "exhaust",
	Doc:      "checks exhaustivity of type switches",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func getType(pass *analysis.Pass, stm ast.Stmt) (types.Type, bool) {
	if assign, ok := stm.(*ast.AssignStmt); ok {
		scrutinee := assign.Rhs[0].(*ast.TypeAssertExpr).X
		return pass.TypesInfo.TypeOf(scrutinee), true
	}
	if expr, ok := stm.(*ast.ExprStmt); ok {
		scrutinee := expr.X.(*ast.TypeAssertExpr).X
		return pass.TypesInfo.TypeOf(scrutinee), true
	}

	return nil, false
}

func run(pass *analysis.Pass) (interface{}, error) {
	i := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.TypeSwitchStmt)(nil),
	}

	i.Preorder(nodeFilter, func(node ast.Node) {
		tySwitch := node.(*ast.TypeSwitchStmt)

		if acquiredType, ok := getType(pass, tySwitch.Assign); ok {
			// the acquired type being nil means it's the boxed interface{} type
			// which doesn't make sense to check for exhaustive matching
			// FIXME: this doesn't accurately check that it's boxed interface type, above reasoning is wrong
			if acquiredType == nil {
				return
			}

			pass.Reportf(tySwitch.Pos(), "should not have reached")

			var clauseTypes []types.Type

			for _, stmt := range tySwitch.Body.List {
				clause := stmt.(*ast.CaseClause)
				name := clause.List[0].(*ast.Ident)
				clauseTy := pass.TypesInfo.TypeOf(name)
				clauseTypes = append(clauseTypes, clauseTy)
			}
		}
	})

	return nil, nil
}
