package commands

import "strings"

// ProjectListSentinel is the --project value when -p is passed without a name (list only).
const ProjectListSentinel = "*"

// mightBeTrailingProjectArg reports whether the final positional could be a project
// name left over from "-p NAME" parsed as bare -p (NoOptDefVal) plus separate NAME.
func mightBeTrailingProjectArg(args []string) bool {
	if project != ProjectListSentinel || len(args) == 0 {
		return false
	}
	last := args[len(args)-1]
	return !strings.Contains(last, "=") && !strings.HasPrefix(last, "-")
}

// resolveTrailingProjectName fixes "-p NAME" / "--project NAME" when cobra parsed the flag
// without a value (NoOptDefVal) and NAME arrived as a separate positional.
func resolveTrailingProjectName(args []string) []string {
	if !mightBeTrailingProjectArg(args) {
		return args
	}
	project = args[len(args)-1]
	return args[:len(args)-1]
}

// trimResolvedProjectArg removes a trailing positional that matches an already-resolved
// project name (e.g. after PersistentPreRunE resolved -p geo).
func trimResolvedProjectArg(args []string) []string {
	if project == "" || project == ProjectListSentinel || len(args) == 0 {
		return args
	}
	last := args[len(args)-1]
	if last == project && !strings.Contains(last, "=") && !strings.HasPrefix(last, "-") {
		return args[:len(args)-1]
	}
	return args
}

// commandArgs returns args with a stray trailing project positional removed.
func commandArgs(args []string) []string {
	args = resolveTrailingProjectName(args)
	return trimResolvedProjectArg(args)
}
