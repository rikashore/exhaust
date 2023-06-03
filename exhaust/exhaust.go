package exhaust

import (
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/exp/slices"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"strings"
)

var Analyzer = &analysis.Analyzer{
	Name:     "exhaust",
	Doc:      "checks exhaustivity of type switches",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

var DefaultExprType = types.NewInterfaceType([]*types.Func{
	types.NewFunc(
		token.NoPos,
		nil,
		"defaultCase",
		types.NewSignatureType(
			nil,
			nil,
			nil,
			nil,
			types.NewTuple(
				types.NewVar(
					token.NoPos,
					nil, "foo",
					types.Universe.Lookup("error").Type(),
				),
			),
			false,
		),
	),
}, nil)

func init() {
	Analyzer.Flags.Bool("ignore-nil", false, "check for exhaustive match even with nil or default case")
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
			// It doesn't make sense to check for an exhaustive match
			// if the type of is the boxed interface type
			// Named interfaces are of type types.Named
			if _, ok := acquiredType.(*types.Interface); ok {
				return
			}

			var clauseTypes []types.Type

			for _, stmt := range tySwitch.Body.List {
				clause := stmt.(*ast.CaseClause)

				if clause.List == nil {
					clauseTypes = append(clauseTypes, DefaultExprType)
					continue
				}

				name := clause.List[0].(*ast.Ident)
				clauseTy := pass.TypesInfo.TypeOf(name)
				clauseTypes = append(clauseTypes, clauseTy)
			}

			// checks if a `default` or case checking nil is present
			hasDefaultOrNilCase := func(t types.Type) bool {
				if types.IdenticalIgnoreTags(t, DefaultExprType) {
					return true
				}

				if basic, ok := t.(*types.Basic); ok {
					return basic.Kind() == 25
				}
				return false
			}

			nilFlag := pass.Analyzer.Flags.Lookup("ignore-nil")
			ignoreNil := nilFlag.Value.String()

			// Only exit if a default case or nil case is handled
			// and that we should *not* ignore the nil case
			if slices.ContainsFunc(clauseTypes, hasDefaultOrNilCase) && ignoreNil == "false" {
				return
			}

			sc := pass.Pkg.Scope()
			underlyingInterface := acquiredType.Underlying().(*types.Interface)

			var implementations []types.Type

			for _, objName := range sc.Names() {
				obj := sc.Lookup(objName)
				objTy := obj.Type()
				if types.Implements(objTy, underlyingInterface) && !types.IsInterface(objTy) {
					implementations = append(implementations, objTy)
				}
			}

			var unmatched []types.Type

			for _, ty := range implementations {
				idx := slices.Index(clauseTypes, ty)
				if idx == -1 {
					unmatched = append(unmatched, ty)
				}
			}

			if len(unmatched) == 0 {
				return
			}

			builder := strings.Builder{}

			for _, unmatchedTy := range unmatched {
				builder.WriteString("- ")
				builder.WriteString(types.TypeString(unmatchedTy, types.RelativeTo(pass.Pkg)))
				builder.WriteString("\n")
			}

			pass.Reportf(
				tySwitch.Pos(),
				"Inexhaustive pattern match for %s, missing cases\n%s",
				types.TypeString(acquiredType, types.RelativeTo(pass.Pkg)), builder.String(),
			)
		}
	})

	return nil, nil
}
