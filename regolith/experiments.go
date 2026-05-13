package regolith

import "slices"

type Experiment int

const (
	SizeTimeCheck Experiment = iota
	SymlinkExport
)

type ExperimentInfo struct {
	Name        string
	Description string
}

var AvailableExperiments = map[Experiment]ExperimentInfo{}

var EnabledExperiments []string

func IsExperimentEnabled(exp Experiment) bool {
	if EnabledExperiments == nil {
		return false
	}
	return slices.Contains(EnabledExperiments, AvailableExperiments[exp].Name)
}
