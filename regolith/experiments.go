package regolith

type Experiment int

const (
	// SizeTimeCheck is an experiment that checks the size and modification time when exporting
	SizeTimeCheck Experiment = iota
)

var experimentNames = map[Experiment]string{
	SizeTimeCheck: "size_time_check",
}

var EnabledExperiments []string

func IsExperimentEnabled(exp Experiment) bool {
	if EnabledExperiments == nil {
		return false
	}
	for _, e := range EnabledExperiments {
		if e == experimentNames[exp] {
			return true
		}
	}
	return false
}
