package regolith

import (
	"runtime"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/stirante/go-simple-eval/eval"
	"github.com/stirante/go-simple-eval/eval/utils"
)

func EvalCondition(condition string, ctx RunContext) (bool, error) {
	Logger.Debugf("Evaluating condition: %s", condition)
	t := prepareScope(ctx)
	Logger.Debugf("Evaluation scope: %s", utils.ToString(t))
	e, err := eval.Eval(condition, t)
	if err != nil {
		return false, burrito.WrapErrorf(err, "Failed to evaluate condition: %s", condition)
	}
	Logger.Debugf("Condition evaluated to: %s", utils.ToString(e))
	return utils.ToBoolean(e), nil
}

func prepareScope(ctx RunContext) map[string]interface{} {
	semverString, err := utils.ParseSemverString(Version)
	if err != nil {
		semverString = utils.Semver{}
	}
	return map[string]interface{}{
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"debug":          burrito.Debug,
		"version":        semverString,
		"profile":        ctx.Profile,
		"filterLocation": ctx.AbsoluteLocation,
		"isTool":         ctx.IsTool,
	}
}
