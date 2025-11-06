package regolith

import "github.com/Bedrock-OSS/go-burrito/burrito"

type Experiment int

const (
	// SizeTimeCheck is an experiment that checks the size and modification time when exporting
	SizeTimeCheck Experiment = iota
	// SymlinkExport links the temporary build directory with the export
	// target using hard links when possible.
	SymlinkExport
)

// The descriptions shouldn't be too wide, the text with their description is
// indented a lot.
const sizeTimeCheckDesc = `
Activates optimization for file exporting by checking the size and
modification time of files before exporting, and only exporting if
the file has changed. This experiment applies to 'run' and 'watch'
commands.
`

const symlinkExportDesc = `
Creates links from the tmp directory to the export target so that files
written to tmp are immediately reflected in the export location.`

type ExperimentInfo struct {
	Name        string
	Description string
}

var AvailableExperiments = map[Experiment]ExperimentInfo{
	SizeTimeCheck: {"size_time_check", sizeTimeCheckDesc},
	SymlinkExport: {"symlink_export", symlinkExportDesc},
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

func ValidateExperiments() error {
	sizeTimeCheckEnabled := IsExperimentEnabled(SizeTimeCheck)
	symlinkExportEnabled := IsExperimentEnabled(SymlinkExport)
	if sizeTimeCheckEnabled && symlinkExportEnabled {
		return burrito.WrappedError("size_time_check and symlink_export cannot be enabled at the same time")
	}
	return nil
}
