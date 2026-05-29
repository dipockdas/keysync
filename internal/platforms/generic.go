package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/dipockdas/keysync/internal/config"
	"github.com/dipockdas/keysync/internal/store"
)

// GenericPlatform implements the Platform interface using declarative CLI or HTTP configs.
// This allows users to add support for any platform without writing Go code.
type GenericPlatform struct {
	name   string
	config *config.GenericPlatformConfig
	token  string
	client HTTPClient
}

// NewGeneric creates a GenericPlatform from a JSON config string.
func NewGeneric(ctx context.Context, name, configJSON string, secretSt store.Store) (Platform, error) {
	var cfg config.GenericPlatformConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("parse generic config for %s: %w", name, err)
	}

	// Validate config
	if cfg.Type != "cli" && cfg.Type != "http" {
		return nil, fmt.Errorf("generic platform %s: type must be 'cli' or 'http', got %q", name, cfg.Type)
	}
	if cfg.TokenEnv == "" {
		return nil, fmt.Errorf("generic platform %s: token_env is required", name)
	}

	// Validate CLI config
	if cfg.Type == "cli" {
		if cfg.Command == "" {
			return nil, fmt.Errorf("generic platform %s: command is required for CLI type", name)
		}
		// Run validation check if specified
		if cfg.Validation != nil && cfg.Validation.CommandCheck != "" {
			if err := validateCLIAvailable(cfg.Validation.CommandCheck); err != nil {
				return nil, fmt.Errorf("generic platform %s: %w", name, err)
			}
		}
	}

	// Validate HTTP config
	if cfg.Type == "http" {
		if cfg.Endpoint == "" {
			return nil, fmt.Errorf("generic platform %s: endpoint is required for HTTP type", name)
		}
		if cfg.Method == "" {
			return nil, fmt.Errorf("generic platform %s: method is required for HTTP type", name)
		}
	}

	// Lookup token from secret store or environment
	token := lookupToken(ctx, secretSt, name)
	if token == "" {
		// Try token_env as the key name
		if secretSt != nil {
			token, _ = secretSt.Get(ctx, "global", "", "", cfg.TokenEnv)
		}
		if token == "" {
			token = osGetenv(cfg.TokenEnv)
		}
	}
	if token == "" {
		return nil, fmt.Errorf("generic platform %s: token not found (looked for %s in keychain and environment)", name, cfg.TokenEnv)
	}

	return &GenericPlatform{
		name:   name,
		config: &cfg,
		token:  token,
		client: &http.Client{
			Timeout: 30 * time.Second, // Prevent indefinite hangs (Finding 6)
		},
	}, nil
}

// Name returns the platform name.
func (g *GenericPlatform) Name() string {
	return g.name
}

// Upsert creates or updates a secret on the platform.
func (g *GenericPlatform) Upsert(ctx context.Context, key, value string) error {
	// Validate key name (security: prevent command injection)
	if !isValidKeyName(key) {
		return fmt.Errorf("invalid key name %q: must contain only A-Z, 0-9, and underscore", key)
	}

	switch g.config.Type {
	case "cli":
		return g.execCLI(ctx, key, value)
	case "http":
		return g.execHTTP(ctx, key, value)
	default:
		return fmt.Errorf("unsupported type: %s", g.config.Type)
	}
}

// execCLI executes a CLI command with template substitution.
func (g *GenericPlatform) execCLI(ctx context.Context, key, value string) error {
	// Build substitution map
	subs := map[string]string{
		"KEY":   key,
		"VALUE": value,
		"TOKEN": g.token, // Standard {TOKEN} placeholder
	}
	// Add token (use token_env name as placeholder for backward compat)
	if g.config.TokenEnv != "" {
		subs[g.config.TokenEnv] = g.token
	}
	// Add template_vars fields
	for k, v := range g.config.TemplateVars {
		subs[k] = v
	}
	// Add config fields (deprecated, for backward compat)
	for k, v := range g.config.Config {
		subs[k] = v
	}

	// Substitute command
	command := replaceTemplates(g.config.Command, subs)

	// Security: validate no shell metacharacters in final command
	if containsShellMetachars(command) {
		return fmt.Errorf("command contains shell metacharacters after substitution (security check failed)")
	}

	// Parse command into name and args
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command after substitution")
	}
	cmdName := parts[0]
	cmdArgs := parts[1:]

	// Create command with context for timeout support (Finding 6)
	cmd := exec.CommandContext(ctx, cmdName, cmdArgs...)

	// Handle stdin if specified
	if g.config.Stdin != "" {
		stdinValue := replaceTemplates(g.config.Stdin, subs)
		cmd.Stdin = strings.NewReader(stdinValue)
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	if err := cmd.Run(); err != nil {
		// Sanitize error output (don't leak secrets)
		errMsg := sanitizeResponseBody([]byte(stderr.String()))
		return fmt.Errorf("command failed: %w (stderr: %s)", err, errMsg)
	}

	return nil
}

// execHTTP executes an HTTP request with template substitution.
func (g *GenericPlatform) execHTTP(ctx context.Context, key, value string) error {
	// Build substitution map
	subs := map[string]string{
		"KEY":   key,
		"VALUE": value,
		"TOKEN": g.token, // Standard {TOKEN} placeholder
	}
	// Add token (use token_env name as placeholder for backward compat)
	if g.config.TokenEnv != "" {
		subs[g.config.TokenEnv] = g.token
	}
	// Add template_vars fields
	for k, v := range g.config.TemplateVars {
		subs[k] = v
	}
	// Add config fields (deprecated, for backward compat)
	for k, v := range g.config.Config {
		subs[k] = v
	}

	// Substitute endpoint
	endpoint := replaceTemplates(g.config.Endpoint, subs)

	// Substitute and marshal body
	var bodyReader io.Reader
	if g.config.Body != nil {
		bodyJSON, err := substituteJSON(g.config.Body, subs)
		if err != nil {
			return fmt.Errorf("substitute body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyJSON)
	}

	// Create request with context for timeout support (Finding 6)
	req, err := http.NewRequestWithContext(ctx, g.config.Method, endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Add headers with substitution
	for k, v := range g.config.Headers {
		headerValue := replaceTemplates(v, subs)
		req.Header.Set(k, headerValue)
	}

	// Set default Content-Type if not provided
	if req.Header.Get("Content-Type") == "" && bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		sanitized := sanitizeResponseBody(body)
		return fmt.Errorf("http %d: %s", resp.StatusCode, sanitized)
	}

	return nil
}

// replaceTemplates replaces {PLACEHOLDER} with values from the substitution map.
func replaceTemplates(template string, subs map[string]string) string {
	result := template
	for key, value := range subs {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// substituteJSON performs template substitution in JSON structures.
// It marshals the structure, does string replacement, then unmarshals back.
func substituteJSON(obj interface{}, subs map[string]string) ([]byte, error) {
	// Marshal to JSON
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	// Replace templates in JSON string
	jsonStr := string(data)
	for key, value := range subs {
		placeholder := "{" + key + "}"
		// Escape JSON special characters in the replacement value
		jsonValue := strings.ReplaceAll(value, "\\", "\\\\")
		jsonValue = strings.ReplaceAll(jsonValue, "\"", "\\\"")
		jsonStr = strings.ReplaceAll(jsonStr, placeholder, jsonValue)
	}

	return []byte(jsonStr), nil
}

// isValidKeyName checks if a key name contains only safe characters.
// Allows: A-Z, a-z, 0-9, underscore
func isValidKeyName(key string) bool {
	matched, _ := regexp.MatchString(`^[A-Za-z0-9_]+$`, key)
	return matched
}

// containsShellMetachars checks if a string contains shell metacharacters.
// This prevents command injection attacks.
func containsShellMetachars(s string) bool {
	// Dangerous characters: ; & | $ ` ( ) < > \ " '
	dangerous := []string{";", "&", "|", "$", "`", "(", ")", "<", ">", "\\", "\"", "'"}
	for _, char := range dangerous {
		if strings.Contains(s, char) {
			return true
		}
	}
	return false
}

// validateCLIAvailable checks if a CLI tool is available by running a validation command.
func validateCLIAvailable(checkCommand string) error {
	parts := strings.Fields(checkCommand)
	if len(parts) == 0 {
		return fmt.Errorf("empty validation command")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("CLI validation failed: %w (is %s installed?)", err, parts[0])
	}
	return nil
}
