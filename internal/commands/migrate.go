package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dipockdas/keysync/internal/config"
	"github.com/dipockdas/keysync/internal/store"
	"github.com/spf13/cobra"
)

var (
	migrateFile   string
	migrateCloud  string
	migrateDryRun bool
)

// pullCloudVars pulls environment variables from a deployment platform CLI.
func pullCloudVars(platform string) ([]parsedKey, error) {
	switch strings.ToLower(platform) {
	case "vercel":
		return pullVercelVars()
	case "railway":
		return pullRailwayVars()
	case "supabase":
		return pullSupabaseVars()
	default:
		return nil, fmt.Errorf("unsupported cloud platform: %q (supported: vercel, railway, supabase)", platform)
	}
}

// pullVercelVars uses `vercel env pull` to a temp file, then parses it as a .env file.
func pullVercelVars() ([]parsedKey, error) {
	tmpFile, err := os.CreateTemp("", "keysync-vercel-*.env")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command("vercel", "env", "pull", "--yes", tmpPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("vercel env pull: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	return parseEnvFile(tmpPath)
}

// pullRailwayVars uses `railway variables` which outputs JSON key-value pairs.
func pullRailwayVars() ([]parsedKey, error) {
	cmd := exec.Command("railway", "variables")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("railway variables: %w", err)
	}

	var vars map[string]string
	if err := json.Unmarshal(out, &vars); err != nil {
		return nil, fmt.Errorf("parse railway variables (expected JSON): %w", err)
	}

	var secrets []parsedKey
	for k, v := range vars {
		secrets = append(secrets, parsedKey{key: k, value: v})
	}
	return secrets, nil
}

// pullSupabaseVars uses `supabase secrets list` with --json flag, falling back to table parsing.
func pullSupabaseVars() ([]parsedKey, error) {
	// Try JSON output first (newer supabase CLI versions)
	cmd := exec.Command("supabase", "secrets", "list", "--json")
	out, err := cmd.Output()
	if err == nil {
		var entries []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}
		if json.Unmarshal(out, &entries) == nil {
			var secrets []parsedKey
			for _, e := range entries {
				secrets = append(secrets, parsedKey{key: e.Name, value: e.Value})
			}
			return secrets, nil
		}
	}

	// Fallback: parse table output
	cmd = exec.Command("supabase", "secrets", "list")
	out, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("supabase secrets list: %w", err)
	}

	return parseSupabaseTable(string(out))
}

// parseSupabaseTable parses the table output of `supabase secrets list`.
func parseSupabaseTable(output string) ([]parsedKey, error) {
	lines := strings.Split(output, "\n")
	var secrets []parsedKey
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip header, separator, and empty lines
		if line == "" || strings.Contains(line, "Name") || strings.HasPrefix(line, "─") {
			continue
		}
		parts := strings.SplitN(line, "│", 2)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name != "" {
			secrets = append(secrets, parsedKey{key: name, value: value})
		}
	}
	return secrets, nil
}

// migratedKey tracks a key that was migrated from .env to keysync.
type migratedKey struct {
	Key     string
	Scope   string
	Project string
}

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [--file .env] [--cloud vercel|railway|supabase] [--project name]",
		Short: "Import secrets into keysync from an .env file or cloud platform",
		Long: F(`Import secrets into keysync from a local {c}.env{/c} file or directly from
a cloud platform (Vercel, Railway, or Supabase).

For each key found, you interactively choose the scope (global, project, or
project+env) and confirm storage. Use {c}--dry-run{/c} to preview without side effects.

After migration, keysync prints step-by-step instructions for the coding
assistant to complete the migration, with code examples for TypeScript, Go,
and Python.

{b}Examples:{/b}
  {c}keysync migrate --file .env.local{/c}
  {c}keysync migrate --file .env --project my-app{/c}
  {c}keysync migrate --cloud vercel --project my-app{/c}
  {c}keysync migrate --cloud railway --project my-app{/c}
  {c}keysync migrate --cloud supabase --project my-app --dry-run{/c}

{b}See also:{/b}
  Migration docs: {u}https://github.com/dipockdas/keysync#migration{/u}
  Tutorial: {u}https://github.com/dipockdas/keysync/blob/main/docs/tutorial-go-project.md{/u}`),
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			var secrets []parsedKey
			var sourceLabel string

			// Determine source: --cloud takes precedence over --file
			if migrateCloud != "" {
				sourceLabel = fmt.Sprintf("cloud platform: %s", migrateCloud)
				fmt.Printf("Pulling secrets from %s...\n", sourceLabel)
				var err error
				secrets, err = pullCloudVars(migrateCloud)
				if err != nil {
					return fmt.Errorf("pull from %s: %w", migrateCloud, err)
				}
			} else {
				// Determine the file to read
				filePath := migrateFile
				if filePath == "" {
					filePath = ".env"
				}
				sourceLabel = filePath

				// Check the file exists
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					return fmt.Errorf("file not found: %s. Use --file to specify a path or --cloud to pull from a deployment platform", filePath)
				}

				// Parse the .env file
				var err error
				secrets, err = parseEnvFile(filePath)
				if err != nil {
					return fmt.Errorf("parse env file: %w", err)
				}
			}

			if len(secrets) == 0 {
				fmt.Println("No secrets found in", sourceLabel)
				return nil
			}

			fmt.Printf("Found %d secrets in %s\n", len(secrets), sourceLabel)
			if migrateDryRun {
				fmt.Println("🔎 DRY RUN — no secrets will be stored.")
			}
			fmt.Println()

			// Initialize the store (needed since PersistentPreRunE skips migrate)
			initStore()

			// Detect git repo and suggest project for auto-config
			detectedRepo, _ := detectGitRepo()
			detectedName := detectProjectName()
			if detectedRepo != "" {
				fmt.Printf("Detected git repo: %s\n", detectedRepo)
				if project == "" && detectedName != "" {
					fmt.Printf("Suggested project name: %s (use --project to set)\n", detectedName)
				}
			}
			fmt.Println()

			// Interactive migration — remember scope/project between keys
			scanner := bufio.NewScanner(os.Stdin)
			var migratedKeys []migratedKey
			var savedScope string // "", "global", or "project"
			var savedProject string

			// Default project name: from --project flag, or detected from CWD
			defaultProject := project
			if defaultProject == "" {
				defaultProject = detectedName
			}

			for _, kv := range secrets {
				if err := validateKeyName(kv.key); err != nil {
					fmt.Printf("  SKIP: %s (%v)\n", kv.key, err)
					continue
				}
				fmt.Printf("  %s=***\n", kv.key)

				// Scope prompt — use saved choice as default
				def := scopePromptDefaults(savedScope, savedProject)
				fmt.Printf("    Scope: [g]lobal / [p]roject (default: %s): ", def.hint)
				scopeChoice := readLine(scanner, def.key)
				if scanner.Err() != nil {
					return fmt.Errorf("read input: %w", scanner.Err())
				}

				scope := store.ScopeGlobal
				proj := ""
				env := ""
				if strings.EqualFold(scopeChoice, "p") || strings.EqualFold(scopeChoice, "project") {
					scope = store.ScopeProject
					savedScope = "project"
					proj = project
					if proj == "" {
						prompt := fmt.Sprintf("    Project name (default: %s): ", savedProject)
						if savedProject == "" {
							prompt = "    Project name: "
						}
						fmt.Print(prompt)
						proj = readLine(scanner, defaultProject)
						if proj == "" {
							fmt.Println("    Skipping — no project name given.")
							continue
						}
						savedProject = proj
					}
					// Use default env (production) for project-scoped migration
					env = envFlag
				} else {
					savedScope = "global"
				}

				if migrateDryRun {
					fmt.Printf("    [DRY RUN] Would store as %s/%s%s\n", scopeLabel(scope, proj), kv.key, envLabel(env))
				} else {
					// Store prompt
					fmt.Printf("    Store in %s? [Y/n]: ", storeLabel())
					storeChoice := readLine(scanner, "y")
					if scanner.Err() != nil {
						return fmt.Errorf("read input: %w", scanner.Err())
					}

					if strings.EqualFold(storeChoice, "n") || strings.EqualFold(storeChoice, "no") {
						fmt.Println("    Skipped.")
						continue
					}

					// Store the secret
					if err := secretSt.Set(cobraCmd.Context(), scope, proj, env, kv.key, kv.value); err != nil {
						fmt.Fprintf(os.Stderr, "    Error: %v\n", err)
						continue
					}

					fmt.Printf("    Stored as %s/%s%s\n", scopeLabel(scope, proj), kv.key, envLabel(env))
				}

				migratedKeys = append(migratedKeys, migratedKey{
					Key:     kv.key,
					Scope:   string(scope),
					Project: proj,
				})
			}

			// Summary
			summaryHeader := "Migration Complete"
			if migrateDryRun {
				summaryHeader = "DRY RUN Complete"
			}
			fmt.Printf("\n=== %s ===\n", summaryHeader)
			if migrateDryRun {
				fmt.Printf("%d of %d secrets would be migrated.\n", len(migratedKeys), len(secrets))
				fmt.Println("Run without --dry-run to apply.")
			} else {
				fmt.Printf("%d of %d secrets migrated.\n", len(migratedKeys), len(secrets))
			}
			fmt.Println()

			// Auto-create .keysync.json if repo was detected
			if !migrateDryRun && detectedRepo != "" && project != "" {
				fmt.Printf("Repo %q detected with project %q.\n", detectedRepo, project)
				fmt.Print("Save this mapping to .keysync.json? [Y/n]: ")
				saveChoice := readLine(scanner, "y")
				if !strings.EqualFold(saveChoice, "n") && !strings.EqualFold(saveChoice, "no") {
					if err := saveRepoConfig(detectedRepo, project); err != nil {
						fmt.Fprintf(os.Stderr, "  Warning: failed to save .keysync.json: %v\n", err)
					} else {
						fmt.Printf("  ✓ Saved .keysync.json — repo %q → project %q\n", detectedRepo, project)
						fmt.Println("  Edit this file to add platform configurations (Vercel, Railway, Supabase) if needed.")
					}
				}
				fmt.Println()
			}

			if len(migratedKeys) == 0 {
				return nil
			}

			// Print machine-parseable JSON summary for coding assistants
			fmt.Println("---MIGRATION_RESULT_START---")
			fmt.Printf(`{"project":%q,"migrated":[`, project)
			for i, m := range migratedKeys {
				if i > 0 {
					fmt.Printf(",")
				}
				fmt.Printf(`{"key":%q,"scope":%q,"project":%q}`, m.Key, m.Scope, m.Project)
			}
			fmt.Println(`]}`)
			fmt.Println("---MIGRATION_RESULT_END---")
			fmt.Println()

			// Human-readable instructions for the coding assistant
			printMigrationInstructions(migratedKeys)

			// Source code scan for references to migrated keys
			if !migrateDryRun && len(migratedKeys) > 0 {
				scanResults := scanSourceCode(".", migratedKeys)
				printCleanupGuide(".", migratedKeys, scanResults)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&migrateFile, "file", "", "path to .env file (default: .env)")
	cmd.Flags().StringVar(&migrateCloud, "cloud", "", "pull from cloud platform: vercel, railway, or supabase")
	cmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "preview migration without storing secrets")
	return cmd
}

// envLabel returns a human-readable environment suffix.
func envLabel(env string) string {
	if env != "" {
		return fmt.Sprintf(" (env: %s)", env)
	}
	return ""
}

// parsedKey holds a single parsed key=value from the env file.
type parsedKey struct {
	key   string
	value string
}

// parseEnvFile reads a .env file and returns key=value pairs.
// It handles:
//   - Comments (lines starting with #)
//   - Empty lines
//   - Quoted values (' and ")
//   - Export prefix (export KEY=value)
func parseEnvFile(path string) ([]parsedKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var secrets []parsedKey
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip "export " prefix
		line = strings.TrimPrefix(line, "export ")

		// Split on first =
		eq := strings.IndexByte(line, '=')
		if eq < 1 {
			continue
		}

		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])

		// Strip surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if key != "" {
			secrets = append(secrets, parsedKey{key: key, value: value})
		}
	}
	return secrets, nil
}

// initStore initializes the secret store for commands that skip PersistentPreRunE.
func initStore() {
	if secretSt != nil {
		return
	}
	secretSt = store.NewMemoryStore()
	// Try to open the real store
	if st, err := tryKeychain(nil); err == nil {
		secretSt = st
	} else if fs, err := store.NewFallbackStore(); err == nil {
		secretSt = fs
	}
}

// readLine reads a line from stdin, returning the default if input is empty.
func readLine(scanner *bufio.Scanner, defaultVal string) string {
	if !scanner.Scan() {
		return defaultVal
	}
	text := strings.TrimSpace(scanner.Text())
	if text == "" {
		return defaultVal
	}
	return text
}

// scopePromptDef holds the default key and human label for the scope prompt.
type scopePromptDef struct {
	key  string // "g" or "p"
	hint string // "global", "project", "project - myapp"
}

// scopePromptDefaults computes the scope prompt defaults based on previously
// saved choices, enabling "remember between keys" behaviour.
func scopePromptDefaults(savedScope, savedProject string) scopePromptDef {
	if savedScope == "project" {
		hint := "project"
		if savedProject != "" {
			hint += " - " + savedProject
		}
		return scopePromptDef{key: "p", hint: hint}
	}
	return scopePromptDef{key: "g", hint: "global"}
}

// storeLabel returns a human-readable name for the current OS store.
func storeLabel() string {
	// Use type name as a string to avoid platform-specific build constraints
	name := fmt.Sprintf("%T", secretSt)
	switch {
	case strings.Contains(name, "Keychain"):
		return "macOS Keychain"
	case strings.Contains(name, "Fallback"):
		return "encrypted file store"
	case strings.Contains(name, "Memory"):
		return "in-memory store"
	default:
		return "secret store"
	}
}

// detectGitRepo detects the GitHub repo from the current git remote.
// Returns "owner/repo" or empty string if not detectable.
func detectGitRepo() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repo with remote 'origin'")
	}
	return parseGitHubRepo(strings.TrimSpace(string(out))), nil
}

// parseGitHubRepo extracts "owner/repo" from various GitHub URL formats:
//
//	git@github.com:owner/repo.git       → owner/repo
//	https://github.com/owner/repo.git   → owner/repo
//	https://github.com/owner/repo       → owner/repo
func parseGitHubRepo(url string) string {
	url = strings.TrimSuffix(url, ".git")
	// Handle git@github.com:owner/repo
	if before, after, ok := strings.Cut(url, ":"); ok && strings.HasPrefix(before, "git@") {
		return strings.TrimPrefix(after, "/")
	}
	// Handle https://github.com/owner/repo
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		parts := strings.SplitN(url, "/", 4)
		if len(parts) == 4 {
			return parts[3]
		}
	}
	return url
}

// detectProjectName returns the current directory name as a suggested project name.
func detectProjectName() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Base(dir)
}

// saveRepoConfig creates or updates .keysync.json with a repo entry.
// If the file already exists, the repo entry is added to the existing config.
func saveRepoConfig(repo, projectName string) error {
	cfg := config.DefaultConfig()
	if existing, path, err := config.LoadConfig("."); err == nil && path != "" {
		cfg = existing
	}
	cfg.Repos[repo] = config.RepoConfig{
		Project:   projectName,
		Globals:   cfg.Repos[repo].Globals, // preserve any existing globals
		Platforms: make(map[string]json.RawMessage),
	}
	savePath := config.DefaultConfigPath(".")
	return config.SaveConfig(cfg, savePath)
}

// keyReference records a source file reference to a migrated secret key.
type keyReference struct {
	Key      string
	FilePath string
	Line     int
	Content  string
}

// scanSourceCode walks the project directory and finds references to migrated keys
// in source files. It skips .git, node_modules, vendor, and build output directories.
func scanSourceCode(rootDir string, migratedKeys []migratedKey) []keyReference {
	// Build set of keys to search for
	keySet := make(map[string]bool, len(migratedKeys))
	for _, m := range migratedKeys {
		keySet[m.Key] = true
	}

	var refs []keyReference
	var mu sync.Mutex

	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" || base == "vendor" ||
				base == ".next" || base == "dist" || base == "build" ||
				base == "__pycache__" || base == ".venv" || base == "venv" ||
				strings.HasPrefix(base, ".env") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan source-like files
		ext := strings.ToLower(filepath.Ext(path))
		if !isScannableExt(ext) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			for key := range keySet {
				if strings.Contains(line, key) && isEnvReference(line, key) {
					mu.Lock()
					refs = append(refs, keyReference{
						Key:      key,
						FilePath: path,
						Line:     i + 1,
						Content:  strings.TrimSpace(line),
					})
					mu.Unlock()
				}
			}
		}
		return nil
	})

	return refs
}

// isScannableExt returns true for file extensions that may contain env var references.
func isScannableExt(ext string) bool {
	switch ext {
	case ".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".rb", ".sh", ".bash",
		".yaml", ".yml", ".json", ".toml", ".cfg", ".conf", ".ini",
		".env", ".env.example", ".env.sample":
		return true
	}
	return false
}

// isEnvReference checks if a line contains an environment variable reference
// for the given key, reducing false positives from coincidental name matches.
func isEnvReference(line, key string) bool {
	lower := strings.ToLower(line)

	// Check for explicit env var access patterns
	patterns := []string{
		"process.env." + key,
		"process.env[\"" + key,
		"process.env['" + key,
		"process.env[`" + key,
		"os.Getenv(\"" + key,
		"os.Getenv(`" + key,
		"os.LookupEnv(\"" + key,
		"os.LookupEnv(`" + key,
		"os.Environ[\"" + key,
		"os.environ.get(\"" + key,
		"os.environ[\"" + key,
		"os.environ['" + key,
		"ENV[\"" + key,
		"ENV['" + key,
		"ENV.fetch(\"" + key,
		"ENV.fetch('" + key,
		"\"" + key + "\"",   // generic quoted string
		"${" + key + "}",     // shell/bash
	}
	for _, p := range patterns {
		if strings.Contains(line, p) {
			return true
		}
	}

	// Check for lowercased key patterns (e.g., os.getenv("database_url"))
	lowerKey := strings.ToLower(key)
	lowerPatterns := []string{
		"os.getenv(\"" + lowerKey,
		"os.environ[\"" + lowerKey,
		"process.env." + lowerKey,
	}
	for _, p := range lowerPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	return false
}

// printCleanupGuide prints a structured cleanup section after migration,
// listing which keys to remove from .env and which source files reference them.
func printCleanupGuide(rootDir string, migratedKeys []migratedKey, refs []keyReference) {
	fmt.Println()
	fmt.Println("=== Cleanup Guide ===")
	fmt.Println()

	// 1. Keys to remove from .env
	fmt.Println("1. Remove these keys from your .env file (they're now in your OS keychain):")
	for _, m := range migratedKeys {
		fmt.Printf("   - %s\n", m.Key)
	}
	fmt.Println()

	// 2. Source code references
	if len(refs) > 0 {
		fmt.Println("2. Source code references found — update these files to use keysync:")

		// Group references by key
		type groupedRef struct {
			key  string
			refs []keyReference
		}
		keyOrder := make([]string, 0, len(migratedKeys))
		grouped := make(map[string][]keyReference)
		for _, r := range refs {
			if grouped[r.Key] == nil {
				keyOrder = append(keyOrder, r.Key)
			}
			grouped[r.Key] = append(grouped[r.Key], r)
		}

		for _, key := range keyOrder {
			refs := grouped[key]
			fmt.Printf("\n   %s:\n", key)
			for _, r := range refs {
				shortPath := strings.TrimPrefix(r.FilePath, rootDir+string(filepath.Separator))
				if shortPath == r.FilePath {
					shortPath = r.FilePath
				}
				fmt.Printf("     %s:%d  %s\n", shortPath, r.Line, r.Content)
			}
		}
		fmt.Println()

		// 3. Suggested replacements by language
		fmt.Println("3. Suggested replacements — use the keysync client library for your language:")
		printClientLibrarySuggestions(refs)
	} else {
		fmt.Println("2. No source code references found — your project may use .env directly")
		fmt.Println("   without referencing specific keys in code (e.g., dotenv loads all vars).")
		fmt.Println()
		fmt.Println("   Next steps:")
		fmt.Println("    - Replace dotenv/load_dotenv with:")
		fmt.Println("      eval $(keysync export" + projectFlag() + ")")
		fmt.Println("    - Or use the inject approach:")
		fmt.Println("      keysync inject" + projectFlag() + " > .env.local")
	}

	fmt.Println()
	fmt.Println("See the client libraries for language-specific usage:")
	fmt.Println("  https://github.com/dipockdas/keysync/tree/main/clients/")
}

// projectFlag returns " --project <name>" if project is set, empty string otherwise.
func projectFlag() string {
	if project != "" {
		return " --project " + project
	}
	return ""
}

// printClientLibrarySuggestions prints client library usage examples for the
// languages detected in the scan results.
func printClientLibrarySuggestions(refs []keyReference) {
	langs := make(map[string]bool)
	for _, r := range refs {
		ext := strings.ToLower(filepath.Ext(r.FilePath))
		switch ext {
		case ".go":
			langs["Go"] = true
		case ".js", ".ts", ".jsx", ".tsx":
			langs["TypeScript"] = true
		case ".py":
			langs["Python"] = true
		case ".rb":
			langs["Ruby"] = true
		}
	}

	suggestions := map[string]string{
		"Go": `   Go:     keysync.GetSecret("KEY", "` + project + `")
           https://github.com/dipockdas/keysync/blob/main/clients/go/`,
		"TypeScript": `   Node:   getSecret("KEY", "` + project + `")
           https://github.com/dipockdas/keysync/blob/main/clients/node/`,
		"Python": `   Python: get_secret("KEY", project="` + project + `")
           https://github.com/dipockdas/keysync/blob/main/clients/python/`,
		"Ruby": `   Ruby:   KeySync.get_secret("KEY", project: "` + project + `")
           (see clients/ for Go, Python, TypeScript, Swift)`,
	}

	// Print in a consistent order
	langOrder := []string{"Go", "Python", "TypeScript", "Ruby"}
	for _, lang := range langOrder {
		if langs[lang] {
			fmt.Println(suggestions[lang])
		}
	}
	fmt.Println()
	fmt.Println("   For full documentation, see the clients/ directory in the repo.")
}

// printMigrationInstructions outputs concise next-step instructions for the coding
// assistant to complete the migration (replace .env access in source code).
func printMigrationInstructions(keys []migratedKey) {
	fmt.Println("---INSTRUCTIONS_START---")
	fmt.Println("# Migration Complete — Next Steps for Coding Assistant")
	fmt.Println()
	fmt.Println("These secrets are now in the OS keychain. Replace all direct .env access")
	fmt.Println("with keysync calls in the source code below.")
	fmt.Println()
	fmt.Println("## Migrated secrets")
	for _, m := range keys {
		label := "global"
		if m.Scope == "project" {
			label = "project/" + m.Project
		}
		fmt.Printf("- %s → stored as %s\n", m.Key, label)
	}

	fmt.Println()
	fmt.Println("## Replace .env access in source code")
	fmt.Println()
	fmt.Println("### TypeScript / Node.js")
	fmt.Println("```typescript")
	fmt.Println(`import { getSecret } from "@keysync/node";`)
	fmt.Println(`const apiKey = await getSecret("SECRET_NAME");`)
	fmt.Println("```")
	fmt.Println()
	fmt.Println("### Go")
	fmt.Println("```go")
	fmt.Println(`import "github.com/dipockdas/keysync/clients/go"`)
	fmt.Println(`val, err := keysync.GetGlobal("SECRET_NAME")  // or keysync.GetSecret(project, "SECRET_NAME")`)
	fmt.Println("```")
	fmt.Println()
	fmt.Println("### Python")
	fmt.Println("```python")
	fmt.Println(`from keysync import get_secret`)
	fmt.Println(`api_key = get_secret("SECRET_NAME")`)
	fmt.Println("```")

	fmt.Println()
	fmt.Println("## Cleanup checklist")
	fmt.Println("1. Remove .env from source control: `echo \".env*\" >> .gitignore`")
	fmt.Println("2. Remove dotenv/config imports from entry points")
	fmt.Println("3. Push secrets to GitHub and platforms: run `keysync push --project <name>` locally")

	fmt.Println("---INSTRUCTIONS_END---")
}
