package regolith

import "github.com/Bedrock-OSS/go-burrito/burrito"

type ProfileFilter struct {
	Filter
	Profile string `json:"-"`
}

func (f *ProfileFilter) Run(context RunContext) (bool, error) {
	Logger.Infof("Running %q nested profile...", f.Profile)
	return RunProfileImpl(RunContext{
		Profile:             f.Profile,
		AbsoluteLocation:    context.AbsoluteLocation,
		Config:              context.Config,
		Parent:              &context,
		interruptionChannel: context.interruptionChannel,
		DotRegolithPath:     context.DotRegolithPath,
		Settings:            f.Settings,
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
