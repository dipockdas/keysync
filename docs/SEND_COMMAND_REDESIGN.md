# Send Command Redesign

## Overview

Refactor `keysync push` → `keysync send` with three major improvements:

1. **Rename sync to send** - More accurate terminology
2. **Make GitHub consistent** - Treat as a platform, not special-cased
3. **Declarative platform support** - Add platforms via config, not code

## Part 1: Rename sync → send

**Rationale**: We're sending (pushing) secrets to platforms, not synchronizing (bidirectional) with them.

### Changes Required

#### Code
- `internal/commands/sync.go` → `internal/commands/send.go`
- Update command registration in `internal/commands/root.go`
- Update help text and examples

#### Documentation
- README.md (all `sync` references → `send`)
- docs/tutorial-go-project.md
- docs/tutorial-windows-setup.md
- docs/migration-guide.md
- .agents/skills/keysync-setup/SKILL.md
- AGENTS.md
- CLAUDE.md
- CONTRIBUTING.md

#### Client Library Docs
- All client README files mention `keysync push`

### Migration Path
- Keep `sync` as an alias to `send` with deprecation warning (6 months)
- Users can use either command during transition

---

## Part 2: GitHub as a Platform

**Rationale**: GitHub Secrets is just another destination. Special-casing it creates inconsistency.

### Current Behavior
```bash
keysync push --project my-app --platforms vercel,railway
# Always pushes to GitHub + only specified platforms
```

### New Behavior
```bash
keysync send --project my-app --platforms github,vercel,railway
# Only pushes to specified platforms (explicit is better)

keysync send --project my-app
# Pushes to ALL configured platforms (including GitHub if configured)
```

### Implementation

1. **Move GitHub to platforms package**
   ```
   internal/github/client.go → internal/platforms/github.go
   ```

2. **Register GitHub like other platforms**
   ```go
   func init() {
       platforms.Register("github", NewGitHub)
   }
   ```

3. **Add to TokenEnvNames**
   ```go
   "github": "GITHUB_TOKEN"  // or "GH_TOKEN"
   ```

4. **Update config.go**
   ```go
   type PlatformConfig struct {
       GitHub   *GitHubConfig   `json:"github,omitempty"`
       Vercel   *VercelConfig   `json:"vercel,omitempty"`
       Railway  *RailwayConfig  `json:"railway,omitempty"`
       Supabase *SupabaseConfig `json:"supabase,omitempty"`
       // Generic platforms (see Part 3)
       Generic  map[string]*GenericPlatformConfig `json:"generic,omitempty"`
   }

   type GitHubConfig struct {
       Repo string `json:"repo"` // Already in repos key, could be redundant
   }
   ```

5. **Remove special handling from send.go**
   - Delete `syncToGitHub()` function
   - Remove lines 91-94 (unconditional GitHub sync)
   - GitHub becomes part of the platform loop

### Breaking Change Mitigation

**Option A**: Auto-add GitHub if not specified
```go
if !contains(platformNames, "github") && cfg has github config {
    platformNames = append([]string{"github"}, platformNames...)
}
```

**Option B**: Deprecation warning
```bash
keysync send --project my-app --platforms vercel
# WARNING: GitHub not specified. Add 'github' to --platforms or it won't sync.
# To sync to GitHub: keysync send --project my-app --platforms github,vercel
```

**Recommendation**: Option A for 6 months, then Option B, then full removal of auto-add.

---

## Part 3: Declarative/Agentic Platform Support

**Rationale**: Users shouldn't need to write Go code to add Cloudflare, GitLab, Netlify, etc.

### Design: Generic Platform Adapter

Allow users to define platforms in `.keysync.json` using:
1. **CLI commands** (for platforms with CLIs: flyctl, netlify, cloudflare, wrangler, glab)
2. **HTTP API templates** (for platforms with REST APIs)

### Example: Cloudflare Workers (CLI-based)

```json
{
  "repos": {
    "myorg/myapp": {
      "project": "myapp",
      "platforms": {
        "cloudflare": {
          "type": "cli",
          "command": "wrangler secret put {KEY} --env production",
          "stdin": "{VALUE}",
          "token_env": "CLOUDFLARE_API_TOKEN",
          "validation": {
            "command_check": "wrangler --version"
          }
        }
      }
    }
  }
}
```

### Example: GitLab CI/CD Variables (HTTP-based)

```json
{
  "repos": {
    "myorg/myapp": {
      "project": "myapp",
      "platforms": {
        "gitlab": {
          "type": "http",
          "endpoint": "https://gitlab.com/api/v4/projects/{PROJECT_ID}/variables",
          "method": "POST",
          "headers": {
            "PRIVATE-TOKEN": "{GITLAB_TOKEN}"
          },
          "body": {
            "key": "{KEY}",
            "value": "{VALUE}",
            "protected": false,
            "masked": true
          },
          "token_env": "GITLAB_TOKEN",
          "config": {
            "PROJECT_ID": "12345678"
          }
        }
      }
    }
  }
}
```

### Example: Render (HTTP-based)

```json
{
  "repos": {
    "myorg/myapp": {
      "project": "myapp",
      "platforms": {
        "render": {
          "type": "http",
          "endpoint": "https://api.render.com/v1/services/{SERVICE_ID}/env-vars",
          "method": "PUT",
          "headers": {
            "Authorization": "Bearer {RENDER_API_KEY}"
          },
          "body": [
            {
              "key": "{KEY}",
              "value": "{VALUE}"
            }
          ],
          "token_env": "RENDER_API_KEY",
          "config": {
            "SERVICE_ID": "srv-abc123"
          }
        }
      }
    }
  }
}
```

### Implementation: GenericPlatform

```go
// internal/platforms/generic.go
package platforms

type GenericPlatformConfig struct {
    Type       string                 `json:"type"`        // "cli" or "http"
    Command    string                 `json:"command,omitempty"`     // CLI command template
    Stdin      string                 `json:"stdin,omitempty"`       // What to pipe to stdin
    Endpoint   string                 `json:"endpoint,omitempty"`    // HTTP endpoint
    Method     string                 `json:"method,omitempty"`      // HTTP method
    Headers    map[string]string      `json:"headers,omitempty"`     // HTTP headers
    Body       interface{}            `json:"body,omitempty"`        // HTTP body template
    TokenEnv   string                 `json:"token_env"`             // Token env var name
    Config     map[string]string      `json:"config,omitempty"`      // Platform-specific config
    Validation *ValidationConfig      `json:"validation,omitempty"`  // Pre-flight checks
}

type ValidationConfig struct {
    CommandCheck string `json:"command_check,omitempty"` // Check CLI is installed
}

type GenericPlatform struct {
    name   string
    config *GenericPlatformConfig
}

func (g *GenericPlatform) Name() string { return g.name }

func (g *GenericPlatform) Upsert(key, value string) error {
    switch g.config.Type {
    case "cli":
        return g.execCLI(key, value)
    case "http":
        return g.execHTTP(key, value)
    default:
        return fmt.Errorf("unsupported generic platform type: %s", g.config.Type)
    }
}

func (g *GenericPlatform) execCLI(key, value string) error {
    // Template substitution: {KEY}, {VALUE}, {TOKEN}
    cmd := replaceTemplates(g.config.Command, map[string]string{
        "KEY": key,
        "VALUE": value,
        // ... token from env
    })

    // Execute command with stdin if specified
    // Similar to how we execute `gh` and `security` commands
}

func (g *GenericPlatform) execHTTP(key, value string) error {
    // Template substitution in endpoint, headers, body
    // Make HTTP request
    // Handle errors with sanitizeResponseBody()
}
```

### AI Assistant Workflow

When a user asks: "Add Cloudflare Workers support"

**AI can now**:
1. Search for Cloudflare Workers API docs or CLI docs
2. Generate the JSON config (no Go code needed)
3. Add to `.keysync.json`
4. Test with `keysync send --project myapp --platforms cloudflare`

**Example AI-generated config**:
```json
{
  "platforms": {
    "cloudflare": {
      "type": "cli",
      "command": "wrangler secret put {KEY}",
      "stdin": "{VALUE}",
      "token_env": "CLOUDFLARE_API_TOKEN"
    }
  }
}
```

### Security Considerations

1. **Command injection prevention**:
   - Validate `{KEY}` contains only `[A-Z0-9_]`
   - Never allow arbitrary shell interpolation
   - Use `exec.Command(name, args...)` not `sh -c`

2. **Secret leakage prevention**:
   - Never log command output
   - Mask secrets in error messages (already done in sanitizeResponseBody)
   - Stdin-based secret passing (not command line args)

3. **Configuration validation**:
   - Validate generic platform configs on load
   - Require `token_env` for all generic platforms
   - Reject configs with suspicious shell metacharacters

### Benefits

✅ **Zero code for new platforms** - Edit JSON, not Go
✅ **AI-friendly** - Assistants can generate configs from docs
✅ **User-friendly** - No PR/merge/release cycle to add platforms
✅ **Backward compatible** - Hardcoded platforms still work
✅ **Extensible** - Power users can add any platform

---

## Migration Strategy

### Phase 1: Command Rename (Non-breaking)
- Add `send` command
- Keep `sync` as deprecated alias
- Update docs to use `send`
- Release: v1.X.0

### Phase 2: GitHub as Platform (Breaking)
- Move GitHub to platforms
- Auto-add GitHub for 6 months (backward compat)
- Update examples
- Release: v2.0.0

### Phase 3: Generic Platforms (Additive)
- Add GenericPlatform implementation
- Update config schema
- Add generic platform examples to docs
- Release: v2.1.0

---

## Open Questions

1. **Command name**: `send` or `push`? (`push` is more git-like, `send` is clearer)
2. **GitHub auto-add duration**: 6 months or 12 months?
3. **Generic platform naming**: `"generic": {"cloudflare": {...}}` or flat `"cloudflare": {"type": "generic", ...}`?
4. **HTTP retry logic**: Should generic HTTP platforms auto-retry on 429/500?
5. **Batch support**: Some platforms support batch upsert - worth supporting in generic adapter?

---

## Implementation Checklist

### Part 1: Rename
- [ ] Create `internal/commands/send.go` (copy of sync.go)
- [ ] Update command registration
- [ ] Add `sync` → `send` alias with deprecation warning
- [ ] Update all documentation
- [ ] Update tests

### Part 2: GitHub as Platform
- [ ] Create `internal/platforms/github.go`
- [ ] Register GitHub platform
- [ ] Add GitHubConfig to config.go
- [ ] Remove syncToGitHub() from send.go
- [ ] Add GitHub to platform loop
- [ ] Add backward compat (auto-add)
- [ ] Update tests

### Part 3: Generic Platforms
- [ ] Design GenericPlatformConfig schema
- [ ] Implement GenericPlatform (CLI mode)
- [ ] Implement GenericPlatform (HTTP mode)
- [ ] Add template substitution engine
- [ ] Add security validations
- [ ] Write tests (mock CLI, mock HTTP)
- [ ] Add examples to docs (Cloudflare, GitLab, Netlify, Render)
- [ ] Update AGENTS.md with AI workflow

---

## Example: Final State

**`.keysync.json`**:
```json
{
  "repos": {
    "myorg/myapp": {
      "project": "myapp",
      "globals": ["SENTRY_DSN"],
      "platforms": {
        "github": {
          "repo": "myorg/myapp"
        },
        "vercel": {
          "projectId": "prj_abc123",
          "target": ["production", "preview"]
        },
        "cloudflare": {
          "type": "cli",
          "command": "wrangler secret put {KEY}",
          "stdin": "{VALUE}",
          "token_env": "CLOUDFLARE_API_TOKEN"
        },
        "gitlab": {
          "type": "http",
          "endpoint": "https://gitlab.com/api/v4/projects/12345/variables",
          "method": "POST",
          "headers": {
            "PRIVATE-TOKEN": "{GITLAB_TOKEN}"
          },
          "body": {
            "key": "{KEY}",
            "value": "{VALUE}",
            "masked": true
          },
          "token_env": "GITLAB_TOKEN"
        }
      }
    }
  }
}
```

**Command usage**:
```bash
# Send to all configured platforms
keysync send --project myapp

# Send to specific platforms
keysync send --project myapp --platforms github,vercel

# Send to staging environment
keysync send --project myapp --env staging --platforms cloudflare

# Send only to generic platforms
keysync send --project myapp --platforms cloudflare,gitlab
```
