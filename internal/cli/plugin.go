package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"jevlint/internal/config"
	"jevlint/internal/packs"
)

func executePlugin(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	userCacheDir func() (string, error),
) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, pluginUsage)
		return exitUsageError
	}
	switch args[0] {
	case "install":
		return pluginInstall(args[1:], stdout, stderr, userCacheDir)
	case "list":
		return pluginList(args[1:], stdout, stderr, userCacheDir)
	case "update":
		return pluginUpdate(args[1:], stdout, stderr, userCacheDir)
	case "remove":
		return pluginRemove(args[1:], stdout, stderr)
	case "-h", "--help":
		fmt.Fprint(stderr, pluginUsage)
		return exitSuccess
	default:
		fmt.Fprintf(stderr, "jevlint: unknown plugin command %q\n\n%s", args[0], pluginUsage)
		return exitUsageError
	}
}

func pluginInstall(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	userCacheDir func() (string, error),
) int {
	configPath, specs, exitCode := parsePluginArgs(args, 1, stderr)
	if exitCode != 0 {
		return exitCode
	}
	if len(specs) != 1 {
		fmt.Fprintln(stderr, "jevlint: plugin install requires a pack source")
		return exitUsageError
	}
	cfg, absolute, exitCode := loadProjectFile(configPath, stderr)
	if exitCode != 0 {
		return exitCode
	}
	ref, err := packs.Install(specs[0], userCacheDir)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	cfg.Packs = upsertPack(cfg.Packs, ref)
	if err := config.Write(absolute, cfg); err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	fmt.Fprintf(stdout, "installed %s@%s\n", ref.ID, ref.SHA)
	return exitSuccess
}

func pluginList(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	userCacheDir func() (string, error),
) int {
	configPath, leftover, exitCode := parsePluginArgs(args, 0, stderr)
	if exitCode != 0 {
		return exitCode
	}
	if len(leftover) > 0 {
		fmt.Fprintln(stderr, "jevlint: plugin list does not take arguments")
		return exitUsageError
	}
	cfg, _, exitCode := loadProjectFile(configPath, stderr)
	if exitCode != 0 {
		return exitCode
	}
	userCache, err := userCacheDir()
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	if len(cfg.Packs) == 0 {
		fmt.Fprintln(stdout, "no packs")
		return exitSuccess
	}
	for _, ref := range cfg.Packs {
		status := "missing"
		if _, statErr := os.Stat(filepath.Join(packs.CacheDir(userCache, ref.SHA, ref.ID), packs.ManifestFile)); statErr == nil {
			status = "cached"
		}
		fmt.Fprintf(stdout, "%s  %s  %s  %s\n", ref.ID, ref.SHA, status, ref.Source)
	}
	return exitSuccess
}

func pluginUpdate(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	userCacheDir func() (string, error),
) int {
	configPath, leftover, exitCode := parsePluginArgs(args, 0, stderr)
	if exitCode != 0 {
		return exitCode
	}
	cfg, absolute, exitCode := loadProjectFile(configPath, stderr)
	if exitCode != 0 {
		return exitCode
	}
	ids := leftover
	if len(ids) == 0 {
		for _, ref := range cfg.Packs {
			ids = append(ids, ref.ID)
		}
	}
	for _, id := range ids {
		ref, ok := packByID(cfg.Packs, id)
		if !ok {
			fmt.Fprintf(stderr, "jevlint: unknown pack %q\n", id)
			return exitUsageError
		}
		spec := ref.Source
		if ref.Ref != "" {
			spec += "@" + ref.Ref
		}
		if ref.Path != "" {
			spec += "#" + ref.Path
		}
		updated, err := packs.Install(spec, userCacheDir)
		if err != nil {
			fmt.Fprintf(stderr, "jevlint: %v\n", err)
			return exitUsageError
		}
		updated.ID = ref.ID
		if updated.ID == "" {
			updated.ID = ref.ID
		}
		cfg.Packs = upsertPack(cfg.Packs, updated)
		fmt.Fprintf(stdout, "updated %s@%s\n", updated.ID, updated.SHA)
	}
	if err := config.Write(absolute, cfg); err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	return exitSuccess
}

func pluginRemove(args []string, stdout io.Writer, stderr io.Writer) int {
	configPath, leftover, exitCode := parsePluginArgs(args, 1, stderr)
	if exitCode != 0 {
		return exitCode
	}
	if len(leftover) != 1 {
		fmt.Fprintln(stderr, "jevlint: plugin remove requires a pack id")
		return exitUsageError
	}
	cfg, absolute, exitCode := loadProjectFile(configPath, stderr)
	if exitCode != 0 {
		return exitCode
	}
	next := make([]config.PackRef, 0, len(cfg.Packs))
	found := false
	for _, ref := range cfg.Packs {
		if ref.ID == leftover[0] {
			found = true
			continue
		}
		next = append(next, ref)
	}
	if !found {
		fmt.Fprintf(stderr, "jevlint: unknown pack %q\n", leftover[0])
		return exitUsageError
	}
	cfg.Packs = next
	if err := config.Write(absolute, cfg); err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return exitUsageError
	}
	fmt.Fprintf(stdout, "removed %s\n", leftover[0])
	return exitSuccess
}

func parsePluginArgs(args []string, minPositional int, stderr io.Writer) (string, []string, int) {
	configPath := defaultConfigFile
	positional := make([]string, 0)
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--config":
			if index+1 >= len(args) {
				fmt.Fprintln(stderr, "jevlint: --config requires a path")
				return "", nil, exitUsageError
			}
			index++
			configPath = args[index]
		case len(arg) > 9 && arg[:9] == "--config=":
			configPath = arg[9:]
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) < minPositional {
		fmt.Fprint(stderr, pluginUsage)
		return "", nil, exitUsageError
	}
	return configPath, positional, exitSuccess
}

func loadProjectFile(configPath string, stderr io.Writer) (config.Config, string, int) {
	absolute, err := filepath.Abs(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: resolve config path: %v\n", err)
		return config.Config{}, "", exitUsageError
	}
	cfg, err := config.Load(absolute)
	if err != nil {
		fmt.Fprintf(stderr, "jevlint: %v\n", err)
		return config.Config{}, "", exitUsageError
	}
	return cfg, absolute, exitSuccess
}

func upsertPack(refs []config.PackRef, next config.PackRef) []config.PackRef {
	for index, ref := range refs {
		if ref.ID == next.ID {
			refs[index] = next
			return refs
		}
	}
	return append(refs, next)
}

func packByID(refs []config.PackRef, id string) (config.PackRef, bool) {
	for _, ref := range refs {
		if ref.ID == id {
			return ref, true
		}
	}
	return config.PackRef{}, false
}
