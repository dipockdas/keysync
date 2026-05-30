package commands

import (
	"strings"

	"github.com/spf13/cobra"
)

// resolveEnvironmentForCommand returns the environment for project-scoped operations.
// Omitted --env means project-wide (empty string). Pass --env NAME for environment scope.
func resolveEnvironmentForCommand(cmd *cobra.Command) string {
	if project == "" {
		return ""
	}
	if envFlagChanged(cmd) {
		return envFlag
	}
	return ""
}

// envForGet returns the environment for get/pull: only when --env was explicitly passed.
func envForGet(cmd *cobra.Command) string {
	return explicitEnvFlag(cmd)
}

// envForExport returns the environment for export: only when --env was explicitly passed.
func envForExport(cmd *cobra.Command) string {
	return explicitEnvFlag(cmd)
}

func explicitEnvFlag(cmd *cobra.Command) string {
	if project == "" {
		return ""
	}
	if envFlagChanged(cmd) {
		return envFlag
	}
	return ""
}

func envFlagChanged(cmd *cobra.Command) bool {
	if f := cmd.Flags().Lookup("env"); f != nil && f.Changed {
		return true
	}
	if root := cmd.Root(); root != nil && root != cmd {
		if f := root.PersistentFlags().Lookup("env"); f != nil && f.Changed {
			return true
		}
	}
	return false
}

// resolveEnvironmentFromArgs mirrors resolveEnvironmentForCommand for tests.
func resolveEnvironmentFromArgs(cmdName string, args []string) string {
	if project == "" {
		return ""
	}
	for i, a := range args {
		switch {
		case a == "--env" || a == "-e":
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		case strings.HasPrefix(a, "--env="):
			return strings.TrimPrefix(a, "--env=")
		}
	}
	return ""
}

// applyExplicitEnvFlag marks --env on get/pull when tests pass it on the command line
// (leaf commands do not register the persistent root flag).
func applyExplicitEnvFlag(cmd *cobra.Command, args []string) {
	if cmd.Name() != "get" && cmd.Name() != "pull" {
		return
	}
	for i, a := range args {
		switch {
		case a == "--env" || a == "-e":
			if cmd.Flags().Lookup("env") == nil {
				cmd.Flags().StringVar(&envFlag, "env", "", "")
			}
			val := ""
			if i+1 < len(args) {
				val = args[i+1]
			}
			_ = cmd.Flags().Set("env", val)
			return
		case strings.HasPrefix(a, "--env="):
			if cmd.Flags().Lookup("env") == nil {
				cmd.Flags().StringVar(&envFlag, "env", "", "")
			}
			_ = cmd.Flags().Set("env", strings.TrimPrefix(a, "--env="))
			return
		}
	}
}

// stripEnvFlags removes --env / -e flags from args so leaf commands without the
// persistent flag definition do not fail parsing in tests.
func stripEnvFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--env" || args[i] == "-e":
			if i+1 < len(args) {
				i++
			}
		case strings.HasPrefix(args[i], "--env="):
			continue
		default:
			out = append(out, args[i])
		}
	}
	return out
}
