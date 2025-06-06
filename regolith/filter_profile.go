package regolith

import "github.com/Bedrock-OSS/go-burrito/burrito"

type ProfileFilter struct {
	Filter
	Profile string `json:"-"`
}

func (f *ProfileFilter) Run(context RunContext) (bool, error) {
	Logger.Infof("Running %q nested profile...", f.Profile)
	return RunProfileImpl(RunContext{
		Profile:          f.Profile,
		AbsoluteLocation: context.AbsoluteLocation,
		Config:           context.Config,
		Parent:           &context,
		interruption:     context.interruption,
		DotRegolithPath:  context.DotRegolithPath,
		Settings:         f.Settings,
	})
}

func (f *ProfileFilter) Check(context RunContext) error {
	// Check if the profile exists
	profile, ok := context.Config.Profiles[f.Profile]
	if !ok {
		return burrito.WrappedErrorf("Profile not found.\nProfile: %s", f.Profile)
	}
	// Check if the profile we're nesting wasn't already nested
	parent := context.Parent
	for parent != nil {
		if parent.Profile == f.Profile {
			return burrito.WrappedErrorf(
				"Found circular dependency in the profile."+
					"Profile: %s", f.Profile)
		}
		parent = parent.Parent
	}
	return CheckProfileImpl(
		profile, f.Profile, *context.Config, &context,
		context.DotRegolithPath)
}

func (f *ProfileFilter) IsUsingDataExport(dotRegolithPath string, ctx RunContext) (bool, error) {
	profile := ctx.Config.Profiles[f.Profile]
	for filter := range profile.Filters {
		filter := profile.Filters[filter]
		usingDataPath, err := filter.IsUsingDataExport(dotRegolithPath, ctx)
		if err != nil {
			return false, burrito.WrapErrorf(
				err,
				"Failed to check if profile is using data export.\n"+
					"Profile: %s", f.Profile)
		}
		if usingDataPath {
			return true, nil
		}
	}
	return false, nil
}
