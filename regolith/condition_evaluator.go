package regolith

import (
	"go/constant"
	"go/token"
	"go/types"
	"runtime"
)

func EvalCondition(condition string) (bool, error) {
	Logger.Debugf("Evaluating condition: %s", condition)
	t := preparePackage()
	eval, err := types.Eval(token.NewFileSet(), t, token.NoPos, condition)
	if err != nil {
		return false, WrapErrorf(err, "Failed to evaluate condition: %s", condition)
	}
	if eval.Type != types.Typ[types.Bool] && eval.Type != types.Typ[types.UntypedBool] {
		return false, WrappedErrorf("Condition did not evaluate to a boolean: %s", condition)
	}
	Logger.Debugf("Condition evaluated to: %t", constant.BoolVal(eval.Value))
	return constant.BoolVal(eval.Value), nil
}

func preparePackage() *types.Package {
	t := types.NewPackage("regolith", "regolith")
	addStringConstant(t, "os", runtime.GOOS)
	addStringConstant(t, "arch", runtime.GOARCH)
	addStringConstant(t, "version", Version)
	addBoolConstant(t, "debug", Debug)
	return t
}

func addStringConstant(pkg *types.Package, name, value string) {
	pkg.Scope().Insert(types.NewConst(0, pkg, name, types.Typ[types.String], constant.MakeString(value)))
}

func addBoolConstant(pkg *types.Package, name string, value bool) {
	pkg.Scope().Insert(types.NewConst(0, pkg, name, types.Typ[types.Bool], constant.MakeBool(value)))
}
