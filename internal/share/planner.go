package share

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/dipockdas/keysync/internal/share/ksx"
	"github.com/dipockdas/keysync/internal/store"
)

var (
	ErrProjectRequired = errors.New("share project is required")
	ErrNoSecrets       = errors.New("project has no shareable secrets")
	ErrSecretNotFound  = errors.New("secret not found for sharing")
)

type Plan struct {
	Project string
	Secrets []ksx.Secret
}

type Preview struct {
	Project string
	Scope   string
	Count   int
	Keys    []string
}

func BuildPlan(ctx context.Context, secretStore store.Store, project, key string) (Plan, error) {
	if project == "" {
		return Plan{}, ErrProjectRequired
	}
	if secretStore == nil {
		return Plan{}, fmt.Errorf("build share plan: secret store is unavailable")
	}
	if key != "" {
		secrets, err := selectOne(ctx, secretStore, project, key)
		if err != nil {
			return Plan{}, err
		}
		return Plan{Project: project, Secrets: secrets}, nil
	}

	entries, err := secretStore.List(ctx, store.ScopeProject, project, "")
	if err != nil {
		return Plan{}, fmt.Errorf("list project secrets for sharing: %w", err)
	}
	secrets := make([]ksx.Secret, 0, len(entries))
	for _, entry := range entries {
		if entry.Scope != store.ScopeProject || entry.Project != project {
			continue
		}
		secrets = append(secrets, entryToSecret(entry))
	}
	if len(secrets) == 0 {
		return Plan{}, fmt.Errorf("%w: %q", ErrNoSecrets, project)
	}
	sortProjectSecrets(secrets)
	return Plan{Project: project, Secrets: secrets}, nil
}

func selectOne(ctx context.Context, secretStore store.Store, project, key string) ([]ksx.Secret, error) {
	entries, err := secretStore.List(ctx, store.ScopeProject, project, "")
	if err != nil {
		return nil, fmt.Errorf("list project secrets for sharing: %w", err)
	}
	var matches []ksx.Secret
	for _, entry := range entries {
		if entry.Key == key {
			matches = append(matches, entryToSecret(entry))
		}
	}
	if len(matches) > 0 {
		sortProjectSecrets(matches)
		return matches, nil
	}

	entries, err = secretStore.List(ctx, store.ScopeGlobal, "", "")
	if err != nil {
		return nil, fmt.Errorf("list global secrets for sharing: %w", err)
	}
	for _, entry := range entries {
		if entry.Key == key {
			return []ksx.Secret{entryToSecret(entry)}, nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrSecretNotFound, key)
}

func entryToSecret(entry store.SecretEntry) ksx.Secret {
	return ksx.Secret{
		Name:        entry.Key,
		Scope:       entry.Scope,
		Project:     entry.Project,
		Environment: entry.Environment,
	}
}

func sortProjectSecrets(secrets []ksx.Secret) {
	sort.Slice(secrets, func(i, j int) bool {
		if secrets[i].Environment != secrets[j].Environment {
			return secrets[i].Environment < secrets[j].Environment
		}
		return secrets[i].Name < secrets[j].Name
	})
}

func (p Plan) LoadSecrets(ctx context.Context, secretStore store.Store) ([]ksx.Secret, error) {
	secrets := make([]ksx.Secret, len(p.Secrets))
	for i, selected := range p.Secrets {
		value, err := secretStore.Get(ctx, selected.Scope, selected.Project, selected.Environment, selected.Name)
		if err != nil {
			return nil, fmt.Errorf("read selected secret %q for sharing: %w", selected.Name, err)
		}
		selected.Value = value
		secrets[i] = selected
	}
	return secrets, nil
}

func (p Plan) Preview() Preview {
	keys := make([]string, len(p.Secrets))
	scope := ""
	hasEnv := false
	for i, secret := range p.Secrets {
		keys[i] = formatPreviewKey(secret)
		if secret.Environment != "" {
			hasEnv = true
		}
		currentScope := string(secret.Scope)
		if scope == "" {
			scope = currentScope
		} else if scope != currentScope {
			scope = "mixed"
		}
	}
	if hasEnv && scope == "project" {
		scope = "project + env"
	}
	sort.Strings(keys)
	return Preview{Project: p.Project, Scope: scope, Count: len(keys), Keys: keys}
}

func formatPreviewKey(secret ksx.Secret) string {
	if secret.Environment == "" {
		return secret.Name
	}
	return fmt.Sprintf("%s (env: %s)", secret.Name, secret.Environment)
}

func (p Preview) String() string {
	var output strings.Builder
	fmt.Fprintf(&output, "Project: %s\nScope: %s\nKeys: %d\nValues: hidden\n", p.Project, p.Scope, p.Count)
	if len(p.Keys) > 0 {
		output.WriteString("\nKeys included:\n")
		for _, key := range p.Keys {
			fmt.Fprintf(&output, "- %s\n", key)
		}
	}
	return output.String()
}
