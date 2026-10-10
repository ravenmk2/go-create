package template

import (
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// runExpr runs the compiled when expression against the given option values.
func runExpr(prog *vm.Program, options map[string]any) (any, error) {
	return expr.Run(prog, options)
}
