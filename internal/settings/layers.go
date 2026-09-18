package settings

// This file is the chain itself: which layer is merged when, and where the
// selected profile lands among them. The types live in config.go and the
// profile's own loading in profiles.go; all that is here is the order.

// resolveLayers builds the config from the defaults up: the file, then the
// profile the file or the command line names, then the environment, then the
// flags. The profile getting a pass of its own is what makes one work in every
// checkout — it beats what ~/.kori.yml says — while the environment and the
// flags still beat it field by field, so a machine's own settings survive a
// profile the whole suite shares. file is empty under -no-config, which skips
// ~/.kori.yml rather than the profile a flag or KORI_PROFILE names.
func resolveLayers(system string, file, env, flags Config) (Config, error) {
	resolved := Defaults(system)
	resolved.merge(file)
	if name := selectedProfile(file, env, flags); name != "" {
		if err := ResolveProfile(&resolved, name); err != nil {
			return Config{}, err
		}
	}
	resolved.merge(env)
	resolved.merge(flags)
	return resolved, nil
}

// settingsNoConfig is the same chain with the file layer dropped: -no-config
// skips ~/.kori.yml and nothing else, so a profile the flag or the environment
// names still applies.
func settingsNoConfig(system string, flags, env Config) (Config, error) {
	resolved, err := resolveLayers(system, Config{}, env, flags)
	if err != nil {
		return Config{}, err
	}
	return resolveGates(resolved, flags.GatesFile)
}
