package regolith

import (
	"runtime"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/stirante/go-simple-eval/eval"
	"github.com/stirante/go-simple-eval/eval/utils"
)

// EvalCondition evaluates a condition expression with the given context.
func EvalCondition(expression string, ctx RunContext) (bool, error) {
	Logger.Debugf("Evaluating condition: %s", expression)
	t := prepareScope(ctx)
	Logger.Debugf("Evaluation scope: %s", utils.ToString(t))
	e, err := eval.Eval(expression, t)
	if err != nil {
		return false, burrito.WrapErrorf(err, "Failed to evaluate condition: %s", expression)
	}
	Logger.Debugf("Condition evaluated to: %s", utils.ToString(e))
	return utils.ToBoolean(e), nil
}

// EvalString evaluates an expression with the given context and returns the
// result as a string.
func EvalString(expression string, ctx RunContext) (string, error) {
	Logger.Debugf("Evaluating expression: %s", expression)
	t := prepareScope(ctx)
	Logger.Debugf("Evaluation scope: %s", utils.ToString(t))
	e, err := eval.Eval(expression, t)
	if err != nil {
		return "", burrito.WrapErrorf(err, "Failed to evaluate condition: %s", expression)
	}
	Logger.Debugf("Expression evaluated to: %s", utils.ToString(e))
	if v, ok := e.(string); ok {
		return v, nil
	}
	return "", burrito.WrapErrorf(err, "Expression evaluated to non-string value: %s", expression)
}

func prepareScope(ctx RunContext) map[string]interface{} {
	semverString, err := utils.ParseSemverString(Version)
	if err != nil {
		semverString = utils.Semver{}
	}
	ctx.IsInWatchMode()
	projectData := map[string]interface{}{
		"name":   ctx.Config.Name,
		"author": ctx.Config.Author,
	}
	ctx.IsInWatchMode()
	mode := "build"
	if ctx.IsInWatchMode() {
		mode = "watch"
	}
	return map[string]interface{}{
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"debug":          burrito.PrintStackTrace,
		"version":        semverString,
		"profile":        ctx.Profile,
		"filterLocation": ctx.AbsoluteLocation,
		"settings":       ctx.Settings,
		"project":        projectData,
		"mode":           mode,
		"initial":        ctx.initial,
	}
}
