package regolith

type ProfileFilter struct {
	Filter
	Profile string `json:"-"`
}

func (f *ProfileFilter) Run(context RunContext) (bool, error) {
	Logger.Infof("Running %q nested profile...", f.Profile)
	return WatchProfileImpl(RunContext{
		Profile:             f.Profile,
		AbsoluteLocation:    context.AbsoluteLocation,
		Config:              context.Config,
		Parent:              &context,
		interruptionChannel: context.interruptionChannel,
		DotRegolithPath:     context.DotRegolithPath,
	})
}

func (f *ProfileFilter) Check(context RunContext) error {
	// Check if the profile exists
	profile, ok := context.Config.Profiles[f.Profile]
	if !ok {
		return WrappedErrorf("Profile %s not found", f.Profile)
	}
	// Check if the profile we're nesting wasn't already nested
	parent := context.Parent
	for parent != nil {
		if parent.Profile == f.Profile {
			return WrappedErrorf("Profile %s is circularly defined", f.Profile)
		}
		parent = parent.Parent
	}
	return CheckProfileImpl(
		profile, f.Profile, *context.Config, &context,
		context.DotRegolithPath)
}
