# keysync — Handover

**Last updated:** 2026-06-11  
**Purpose:** Resume work after macOS upgrade / reboot, or pick up an AI-assisted session.  
**Repo:** https://github.com/dipockdas/keysync (public, default branch `main`)  
**Local branch:** `main` (tracks `origin/main`)

---

## 0. Session handoff — paste this to resume with an AI assistant

Copy the block below into a new Cursor chat (or point the assistant at this file):

```
Continuing keysync work from HANDOVER.md (2026-06-10 session).

Done and shipped:
- v1.0.6 released — grouped `keysync ls`, `-g` flag
  https://github.com/dipockdas/keysync/releases/tag/v1.0.6
- Git cleanup: `main` tracks `origin/main` only; `original-keysync` archived on GitHub
- HANDOVER.md updated (bc6f5c7 on main)
- Maintainer machine: `make build` + cp to ~/.local/bin/keysync (v1.0.6 verified)

Shipped in v1.0.7:
- macOS `keysync trust` fix (signed partition lists, unsigned ACL, progress bar)
- `keysync ls -p` → project names + key counts

Backlog:
- Bump Homebrew formula: ./scripts/update-homebrew-formula.sh v1.0.7
- Phase 2 ls ideas: -q quiet mode, --json, duplicate-key hints (see plan doc)

Reference docs:
- Repo: HANDOVER.md, docs/releases/v1.0.6.md
- Obsidian: ~/Documents/personal/Keysync/plan-ls-improvements.md
- Agent skill: .agents/skills/keysync-agent/SKILL.md (never run --unmask in agent sessions)
```

---

## 1. Executive summary

keysync is **public and launch-ready** from a product/docs perspective. Recent work focused on:

- **v1.0.6:** grouped `keysync ls` output, `-g` / `--global` flag, release notes in `docs/releases/`
- macOS keychain trust fixes (v1.0.5)
- Open-source polish (badges, security CI, branch protection)
- **Breaking CLI fix:** no silent `dev` environment default — project-wide scope when `--env` is omitted
- README trust positioning (“no server, no cloud”)
- Doc cleanup (removed internal-only guides)
- Marketing strategy documented in personal folder (not in this repo)

**Published release:** [v1.0.7](https://github.com/dipockdas/keysync/releases/tag/v1.0.7) — macOS trust fix, `ls -p` project names.

**Previous:** [v1.0.6](https://github.com/dipockdas/keysync/releases/tag/v1.0.6) — grouped `ls`, `-g` flag.

---

## 2. Repository state

| Item | Value |
|------|--------|
| Visibility | Public |
| Default branch | `main` |
| **Git remote (only)** | `origin` → https://github.com/dipockdas/keysync.git |
| Local tracking | `main` → `origin/main` |
| Archived (do not use) | [original-keysync](https://github.com/dipockdas/original-keysync) — pre-rewrite history (email in commits); **archived 2026-06-10** |
| Demo | [asciinema.org/a/1162047](https://asciinema.org/a/1162047) embedded in README |
| OpenSSF / CodeQL / Security workflows | Enabled; uploads on public repo |
| Branch protection | `main`: PR required, checks `test`, `govulncheck`, `gitleaks` |
| Private vulnerability reporting | Enabled via API |
| Dependabot security updates | Enabled |

### Git remotes — read this if `git status` looks wrong

**Only push and pull from `origin` (public repo).** The local `original` remote was removed; `original-keysync` is archived on GitHub.

History was rewritten at public launch to strip maintainer email from commits. That old line lives only in the archived repo — it shares **no merge-base** with current `main`, so tracking it showed fake “ahead/behind” counts.

```bash
git remote -v          # should show origin only
git status -sb         # should show main...origin/main
git push origin main   # commits and tags
```

Release notes for each version: `docs/releases/vX.Y.Z.md` (used by the release workflow when present).

### Recent commits (newest first)

| Commit | Summary |
|--------|---------|
| `bc6f5c7` | HANDOVER: single-origin git, archived original-keysync |
| `21fb56c` | Grouped `keysync ls`, `-g` flag, v1.0.6 release notes |
| `6695bca` | Merge PR #3 — v1.0.5 release (macOS keychain trust) |
| `c1d1226` | Fix macOS keychain trust for signed builds |
| `92518a1` | Add HANDOVER.md |
| `4158225` | Restore README asciinema embed |

### Recover deleted maintainer doc (if needed)

`docs/github-repository-settings.md` was removed from the repo (backlog material). Recover from git:

```bash
git show 09f77b2:docs/github-repository-settings.md > ~/Documents/personal/Keysync/github-repository-settings.md
```

---

## 3. Product behavior (important for support & marketing)

### Scopes

| Scope | CLI example |
|-------|-------------|
| Global | `keysync set API_KEY=value` |
| Project-wide | `keysync set -p my-app DB_URL=...` (**no `--env`**) |
| Environment | `keysync set -p my-app DB_URL=... --env production` |

**Omitting `--env` with `-p` is project-wide** — not `dev`. This was a deliberate breaking change (no users shipped yet).

- `set`, `list`, `push`: project-wide only when `--env` omitted; env keys included only with `--env NAME`
- `get`, `export`: env bucket only if `--env` passed; else project-wide → global

Docs: [docs/configuration.md#secret-scopes-and---env](docs/configuration.md), [docs/architecture.md](docs/architecture.md).

### Trust model (use in all outward messaging)

> **No keysync server. No keysync cloud.**  
> Secrets are stored in your OS keychain and never leave your machine unless you run `push` — which calls `gh` and your platform APIs directly. You control exactly what gets sent, and when. If this project disappeared tomorrow, your keychain remains intact and all OS APIs keep working.

Also in README (lines 20–21) and updated `blog-post.md` (personal folder).

### Scope positioning (vs Vault)

> keysync doesn't replace Vault for 500-person teams. It replaces `.env` + copy-paste for builders.

---

## 4. Build, test, release

```bash
cd /Users/dipockdas/Projects/keysync
make build              # macOS/Linux/Windows — output ./bin/keysync
make build-signed       # macOS: Developer ID sign (recommended for keychain)
make test               # internal packages, -race
make test-all           # internal + clients/go
make test-platform      # platform engine tests
```

**macOS after every build or install:**

```bash
keysync trust           # updates keychain ACLs for this binary
```

**Do not** force-push all tags (re-triggers release workflow). Tag only when cutting a real release (e.g. `v1.0.7`).

Release workflow reads `docs/releases/<tag>.md` for GitHub release notes when that file exists.

---

## 5. Docs map (in repo)

| Doc | Purpose |
|-----|---------|
| [README.md](README.md) | Landing page, trust callout, demo, quick start |
| [docs/getting-started.md](docs/getting-started.md) | Three-stage onboarding |
| [docs/configuration.md](docs/configuration.md) | Scopes, `.keysync.json`, `--env` |
| [docs/pushing-secrets.md](docs/pushing-secrets.md) | `keysync push` behavior |
| [docs/troubleshooting.md](docs/troubleshooting.md) | **`keysync trust`** for macOS prompts |
| [docs/tests.md](docs/tests.md) | Unit tests + **CI security checks** table |
| [docs/testing.md](docs/testing.md) | Contributor testing patterns |
| [docs/SECURITY-SCANNING.md](docs/SECURITY-SCANNING.md) | govulncheck, Gitleaks, CodeQL, Scorecard |
| [docs/new-client-library-recipe.md](docs/new-client-library-recipe.md) | All client libs table + recipe |
| [clients/README.md](clients/README.md) | Per-language clients (incl. Dart) |

### Removed from repo (intentional)

| File | Reason |
|------|--------|
| `docs/demo.md` | Internal asciinema recording notes — not user docs |
| `docs/SEND_COMMAND_REDESIGN.md` | Unused design |
| `docs/github-repository-settings.md` | Maintainer checklist → personal/backlog |

`docs/blog-post.md` is **gitignored** — canonical blog draft lives in personal folder (below).

---

## 6. Personal / marketing files (outside repo)

**Location:** `~/Documents/personal/Keysync/`

| File | Role |
|------|------|
| `blog-post.md` | **Canonical blog** for Blogger — updated (push not sync, migrate, trust, asciinema link) |
| `Marketing keysync.md` | Audit notes + Agent 2 distribution plan |
| `marketing-keysync-plan.md` | **Execution plan:** positioning, channels, Hermes agent spec, templates, calendar |
| `HANDOVER.md` | Older handover (2026-05-21) — **stale**; use this repo `HANDOVER.md` instead |

### Blog status

- Ready to publish on Blogger after paste from `blog-post.md`
- Cross-post to Dev.to with canonical URL → Blogger
- Do not publish outdated commands (`keysync sync`, `migrate --from-env`)

---

## 7. Marketing — analysis & feedback (2026-05-29)

Full plan: `~/Documents/personal/Keysync/marketing-keysync-plan.md`. Summary of review:

### Verdict

**Execute the plan.** Tiering is right: AI communities + Stack Overflow + HN first; LinkedIn/Product Hunt deprioritized.

### Pinned messages (use everywhere)

1. Trust blockquote (Section 3 above)  
2. Scope: not Vault — `.env` for builders  
3. AI hook: *You type secrets once. Agents help with everything around them.*  
4. Entry: `keysync migrate` (note Homebrew needs tap first — see README)

### Recommended launch order (if time-limited)

1. Pre-flight: `keysync doctor`, demo link, trust on README, test macOS + one Linux  
2. GitHub topics + Dev.to cross-post  
3. **3–5 days** genuine participation in r/golang, r/devops, or Cursor/Claude communities (**no drive-by promo**)  
4. **HN Show HN** (Tue–Wed ~08:00 US PT) — block 4+ hours for comments  
   - **Title:** `I built keysync so coding agents can configure my apps without seeing my secrets`  
5. Hermes monitoring: **manual first** (10 min/day), automate in week 2  
6. Stack Overflow when Hermes surfaces threads (disclose authorship; fix nested code fences in templates)  
7. awesome-go / LinkedIn after spike cools  

### Adjustments from review

| Topic | Recommendation |
|-------|----------------|
| HN timing | Don’t Show HN on Day 3 after one day of Reddit — allow 3–5 days participation first |
| `brew install` | Use full tap line or lead with GitHub Releases for HN |
| Hermes agent | Don’t block launch on building it — manual digest week 1 |
| SO templates | Fix markdown nesting in template 5.1 before posting |
| Rate limit | Cap **~2 promotional keysync mentions per week** to avoid spam |
| HN comment | Mention Windows + Releases builds preemptively |
| README incident | Only remove `docs/demo.md` link — **keep** asciinema SVG (restored in `4158225`) |

### Metrics that matter (month 1)

GitHub **issues** from real installs > star count. Release download counts by platform. SO answer acceptances. HN technical engagement.

### Channels (tier summary)

| Tier | Channels |
|------|----------|
| 1 | Cursor/Claude communities, Stack Overflow, HN Show HN |
| 2 | GitHub topics, Dev.to, awesome-go, r/golang, r/devops |
| 3 | LinkedIn (network only), skip/de-prioritize Product Hunt |

---

## 8. GitHub & security (already configured)

Configured via CLI (admin bypass on push):

- Branch protection on `main`  
- Private vulnerability reporting  
- Dependabot security updates  

**Not in repo:** branch protection setup steps (removed with `github-repository-settings.md`). Recover from git if needed (Section 2).

**User checks manually:** Code scanning enabled in GitHub UI (Advanced Security).

---

## 9. Maintainer machine (outside repo)

Documented in prior sessions — not in git:

| Setting | Purpose |
|---------|---------|
| `~/.cursor/cli-config.json` → `attributeCommitsToAgent: false` | No Cursor co-author on commits |
| `~/.claude/settings.json` → attribution empty | No Claude co-author |
| `~/.codex/config.toml` → `commit_attribution = ""` | No Codex co-author |
| `~/.bootstrap/git-hooks/commit-msg` + global `core.hooksPath` | Strip AI co-author trailers |

Git history was rewritten at public launch for contributor-graph / email privacy — **do not** force-push all tags. Pre-rewrite history is in archived [original-keysync](https://github.com/dipockdas/original-keysync) (read-only).

---

## 10. After macOS reboot — checklist

1. `cd ~/Projects/keysync && git pull origin main`  
2. `make build-signed` or `make build` → `keysync trust`  
3. `keysync doctor`  
4. Confirm README on GitHub shows trust blockquote + asciinema demo  
5. Resume marketing from `marketing-keysync-plan.md` Section 6 (launch sequencing)  
6. Publish blog on Blogger when ready  

### Not started / backlog

- Hermes opportunity-monitoring agent (spec in marketing plan Section 8)  
- README “AI assistants” paragraph higher on page (marketing audit suggestion)  
- Comparison table on README (marketing audit suggestion)  
- Catalyst / NZ press (optional)  

---

## 11. Quick links

| Resource | URL |
|----------|-----|
| Repo | https://github.com/dipockdas/keysync |
| Releases | https://github.com/dipockdas/keysync/releases |
| Demo | https://asciinema.org/a/1162047 |
| Scorecard | https://scorecard.dev/viewer/?uri=github.com/dipockdas/keysync |
| Go Report Card | https://goreportcard.com/report/github.com/dipockdas/keysync |

---

## 12. Contact / continuity

- Primary development path: `/Users/dipockdas/Projects/keysync`  
- Marketing drafts: `/Users/dipockdas/Documents/personal/Keysync/`  
- Agent transcripts: Cursor project `keysync` (conversation history search via mem-query if needed)

When resuming with an AI assistant, point it at this file and `marketing-keysync-plan.md` for launch context.
