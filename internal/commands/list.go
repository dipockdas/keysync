package commands

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

// ProjectListSentinel is the --project value when -p is passed without a name (list only).
const ProjectListSentinel = "*"

var (
	listUnmask bool
	listGlobal bool
)

// resolveListProjectFlag handles bare -p / --project (project names list) vs -p NAME.
// With NoOptDefVal, cobra sets project to ProjectListSentinel when the flag has no =value;
// a separate-arg name (keysync ls --project hyperdx) arrives as a positional arg instead.
func resolveListProjectFlag(cmdArgs []string) (projectsOnly bool) {
	if project != ProjectListSentinel {
		return false
	}
	if len(cmdArgs) > 0 && !strings.HasPrefix(cmdArgs[0], "-") {
		project = cmdArgs[0]
		return false
	}
	return true
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [--project name] [--env name]",
		Aliases: []string{"ls"},
		Short:   "List all managed secrets",
		Long: F(`Lists secrets in the local OS secret store, grouped by scope.

Without flags, all secrets across every project and scope are shown (grouped).
Use {c}-g{/c} / {c}--global{/c} for global keys only, or together with {c}--project{/c}
to include both global and project keys.

With {c}--project NAME{/c}, only that project's keys are shown (not globals).
Pass {c}-p{/c} or {c}--project{/c} with no name to list project names only.
Use {c}--env NAME{/c} to also include keys for a specific environment.

Use {c}--unmask{/c} to also display secret values (for verification purposes).

{b}Examples:{/b}
  {c}keysync list{/c}  (alias: {c}ls{/c})                 # all secrets, grouped
  {c}keysync list -g{/c}                                  # global keys only
  {c}keysync list -p{/c}                                   # project names only
  {c}keysync list -p my-app{/c}                           # one project's keys
  {c}keysync list -g -p my-app{/c}                        # global + project
  {c}keysync list -p my-app --env production{/c}          # project-wide + env
  {c}keysync list --unmask{/c}                            # show values

{b}See also:{/b}
  {c}keysync get --help{/c}
  Tutorial: {u}https://github.com/dipockdas/keysync#quick-start{/u}`),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if resolveListProjectFlag(args) {
				return printProjectsList(ctx, cmd.OutOrStdout(), listGlobal)
			}

			all, err := collectListEntries(ctx)
			if err != nil {
				return err
			}

			if len(all) == 0 {
				fmt.Println("No secrets found.")
				return nil
			}

			printGroupedList(cmd.OutOrStdout(), all, listUnmask, func(e store.SecretEntry) (string, error) {
				return secretSt.Get(ctx, e.Scope, e.Project, e.Environment, e.Key)
			})
			return nil
		},
	}

	cmd.Flags().BoolVarP(&listGlobal, "global", "g", false, "include global secrets (alone: global only; with --project: global and project)")
	cmd.Flags().BoolVar(&listUnmask, "unmask", false, "show secret values alongside key names")
	return cmd
}

// collectListEntries returns secret entries for the current list flags.
func collectListEntries(ctx context.Context) ([]store.SecretEntry, error) {
	hasProject := project != "" && project != ProjectListSentinel

	switch {
	case !listGlobal && !hasProject:
		entries, err := secretSt.List(ctx, "", "", "")
		if err != nil {
			return nil, fmt.Errorf("list secrets: %w", err)
		}
		return entries, nil

	case listGlobal && !hasProject:
		entries, err := secretSt.List(ctx, store.ScopeGlobal, "", "")
		if err != nil {
			return nil, fmt.Errorf("list global secrets: %w", err)
		}
		return entries, nil

	case !listGlobal && hasProject:
		entries, err := collectListProjectEntries(ctx, project, effectiveEnv)
		if err != nil {
			return nil, fmt.Errorf("list project secrets: %w", err)
		}
		return entries, nil

	default: // listGlobal && hasProject
		globalEntries, err := secretSt.List(ctx, store.ScopeGlobal, "", "")
		if err != nil {
			return nil, fmt.Errorf("list global secrets: %w", err)
		}
		projectEntries, err := collectListProjectEntries(ctx, project, effectiveEnv)
		if err != nil {
			return nil, fmt.Errorf("list project secrets: %w", err)
		}
		return append(globalEntries, projectEntries...), nil
	}
}

// collectListProjectEntries lists project keys without merging env overrides.
// When env is empty, only project-wide keys are returned.
// When env is set, project-wide keys and that environment's keys are returned.
func collectListProjectEntries(ctx context.Context, proj, env string) ([]store.SecretEntry, error) {
	all, err := secretSt.List(ctx, store.ScopeProject, proj, "")
	if err != nil {
		return nil, err
	}
	var out []store.SecretEntry
	for _, e := range all {
		if env == "" {
			if e.Environment == "" {
				out = append(out, e)
			}
			continue
		}
		if e.Environment == "" || e.Environment == env {
			out = append(out, e)
		}
	}
	return out, nil
}

type listSection struct {
	global  bool
	project string
	env     string // empty string means project-wide subgroup
	entries []store.SecretEntry
}

func groupListEntries(entries []store.SecretEntry) []listSection {
	bySection := make(map[string]*listSection)
	var order []string

	for _, e := range entries {
		var key string
		sec := &listSection{entries: []store.SecretEntry{}}
		if e.Scope == store.ScopeGlobal {
			key = "global"
			sec.global = true
		} else {
			key = e.Project + "\x00" + e.Environment
			sec.project = e.Project
			sec.env = e.Environment
		}
		if existing, ok := bySection[key]; ok {
			existing.entries = append(existing.entries, e)
		} else {
			sec.entries = append(sec.entries, e)
			bySection[key] = sec
			order = append(order, key)
		}
	}

	sections := make([]listSection, 0, len(order))
	for _, key := range order {
		sec := bySection[key]
		sortEntriesByKey(sec.entries)
		sections = append(sections, *sec)
	}

	sort.Slice(sections, func(i, j int) bool {
		if sections[i].global != sections[j].global {
			return sections[i].global
		}
		if sections[i].project != sections[j].project {
			return strings.ToLower(sections[i].project) < strings.ToLower(sections[j].project)
		}
		// project-wide (empty env) before named environments
		if sections[i].env == "" && sections[j].env != "" {
			return true
		}
		if sections[i].env != "" && sections[j].env == "" {
			return false
		}
		return strings.ToLower(sections[i].env) < strings.ToLower(sections[j].env)
	})

	return sections
}

func sortEntriesByKey(entries []store.SecretEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Key) < strings.ToLower(entries[j].Key)
	})
}

func keyCountLabel(n int) string {
	if n == 1 {
		return "1 key"
	}
	return fmt.Sprintf("%d keys", n)
}

func printGroupedList(w io.Writer, entries []store.SecretEntry, unmask bool, getValue func(store.SecretEntry) (string, error)) {
	sections := groupListEntries(entries)

	var currentProject string
	for i, sec := range sections {
		if i > 0 {
			prev := sections[i-1]
			if sec.global || prev.global || sec.project != prev.project {
				fmt.Fprintln(w)
			}
		}

		if sec.global {
			fmt.Fprintf(w, "%sGlobal (%s)%s\n", cBold+cGold, keyCountLabel(len(sec.entries)), cReset)
			printSectionEntries(w, sec.entries, unmask, getValue, "  ")
			continue
		}

		if sec.project != currentProject {
			fmt.Fprintf(w, "%sProject: %s%s\n", cBold+cGold, sec.project, cReset)
			currentProject = sec.project
		}

		subLabel := "project-wide"
		if sec.env != "" {
			subLabel = sec.env
		}
		fmt.Fprintf(w, "  %s%s (%s)%s\n", cBold, subLabel, keyCountLabel(len(sec.entries)), cReset)
		printSectionEntries(w, sec.entries, unmask, getValue, "    ")
	}
}

func printSectionEntries(w io.Writer, entries []store.SecretEntry, unmask bool, getValue func(store.SecretEntry) (string, error), indent string) {
	for _, e := range entries {
		if unmask {
			val := "***"
			if v, err := getValue(e); err == nil {
				val = v
			}
			fmt.Fprintf(w, "%s%s%s\t%s\n", indent, colorKey(e.Key), cReset, val)
		} else {
			fmt.Fprintf(w, "%s%s%s\n", indent, colorKey(e.Key), cReset)
		}
	}
}

func colorKey(key string) string {
	if noColor || cGreen == "" {
		return key
	}
	return cGreen + key
}

type projectSummary struct {
	name  string
	count int
}

func collectProjectSummaries(ctx context.Context) ([]projectSummary, error) {
	entries, err := secretSt.List(ctx, store.ScopeProject, "", "")
	if err != nil {
		return nil, fmt.Errorf("list project secrets: %w", err)
	}
	counts := make(map[string]int)
	for _, e := range entries {
		if e.Project != "" {
			counts[e.Project]++
		}
	}
	summaries := make([]projectSummary, 0, len(counts))
	for name, count := range counts {
		summaries = append(summaries, projectSummary{name: name, count: count})
	}
	sort.Slice(summaries, func(i, j int) bool {
		return strings.ToLower(summaries[i].name) < strings.ToLower(summaries[j].name)
	})
	return summaries, nil
}

func printProjectsList(ctx context.Context, w io.Writer, includeGlobal bool) error {
	if includeGlobal {
		globals, err := secretSt.List(ctx, store.ScopeGlobal, "", "")
		if err != nil {
			return fmt.Errorf("list global secrets: %w", err)
		}
		if len(globals) > 0 {
			fmt.Fprintf(w, "%sGlobal (%s)%s\n", cBold+cGold, keyCountLabel(len(globals)), cReset)
			sortEntriesByKey(globals)
			printSectionEntries(w, globals, false, nil, "  ")
			fmt.Fprintln(w)
		}
	}

	summaries, err := collectProjectSummaries(ctx)
	if err != nil {
		return err
	}
	if len(summaries) == 0 {
		fmt.Fprintln(w, "No projects found.")
		return nil
	}

	label := "Projects"
	if len(summaries) == 1 {
		fmt.Fprintf(w, "%s%s (%d project)%s\n", cBold+cGold, label, len(summaries), cReset)
	} else {
		fmt.Fprintf(w, "%s%s (%d projects)%s\n", cBold+cGold, label, len(summaries), cReset)
	}
	for _, s := range summaries {
		fmt.Fprintf(w, "  %s%s (%s)\n", colorKey(s.name), cReset, keyCountLabel(s.count))
	}
	return nil
}
