package regolith

type Experiment int

const (
	// SizeTimeCheck is an experiment that checks the size and modification time when exporting
	SizeTimeCheck Experiment = iota
)

// The descriptions shouldn't be too wide, the text with their description is
// indented a lot.
const sizeTimeCheckDesc = `
Activates optimization for file exporting by checking the size and
modification time of files before exporting, and only exporting if
the file has changed. This experiment applies to 'run' and 'watch'
commands.
`

type ExperimentInfo struct {
	Name        string
	Description string
}

var AvailableExperiments = map[Experiment]ExperimentInfo{
	SizeTimeCheck: {"size_time_check", sizeTimeCheckDesc},
}

var EnabledExperiments []string

func IsExperimentEnabled(exp Experiment) bool {
	if EnabledExperiments == nil {
		return false
	}
	for _, e := range EnabledExperiments {
		if e == AvailableExperiments[exp].Name {
			return true
		}
	}
	return false
}
