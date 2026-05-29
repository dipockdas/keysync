# Shell Completion for keysync

Shell completion (also called tab completion or autocomplete) enables you to press Tab to autocomplete keysync commands, flags, and values.

## What gets autocompleted

- **Commands**: `keysync s`<Tab> → `keysync set`
- **Flags**: `keysync set --pro`<Tab> → `keysync set --project`
- **Flag values**: `keysync --store `<Tab> → `keysync --store fallback`

This makes keysync faster to use and reduces typos.

## Supported Shells

Keysync provides completion scripts for:
- **PowerShell** (Windows, macOS, Linux)
- **bash** (macOS, Linux)
- **zsh** (macOS default since Catalina, Linux)
- **fish** (macOS, Linux)

## Quick Setup

### PowerShell

**Current session** (temporary):
```powershell
keysync completion powershell | Out-String | Invoke-Expression
```

**Persistent** (survives restarts):
```powershell
# Find your PowerShell profile location
$PROFILE

# Open it in notepad
notepad $PROFILE

# Add this line:
keysync completion powershell | Out-String | Invoke-Expression
```

**Alternative** (generate to file):
```powershell
# Create config directory if it doesn't exist
New-Item -ItemType Directory -Force -Path "$HOME\.config\keysync"

# Generate completion script
keysync completion powershell > "$HOME\.config\keysync\completion.ps1"

# Add to profile:
# . $HOME\.config\keysync\completion.ps1
```

### bash

**Current session** (temporary):
```bash
source <(keysync completion bash)
```

**Persistent** (survives restarts):
```bash
# Add to ~/.bashrc (Linux) or ~/.bash_profile (macOS)
echo 'source <(keysync completion bash)' >> ~/.bashrc
```

**Alternative** (generate to file):
```bash
# Create config directory
mkdir -p ~/.config/keysync

# Generate completion script
keysync completion bash > ~/.config/keysync/completion.bash

# Add to ~/.bashrc:
echo 'source ~/.config/keysync/completion.bash' >> ~/.bashrc
```

### zsh

**Current session** (temporary):
```zsh
source <(keysync completion zsh)
```

**Persistent** (survives restarts):
```zsh
# Add to ~/.zshrc
echo 'source <(keysync completion zsh)' >> ~/.zshrc
```

**Alternative** (generate to file):
```zsh
# Create config directory
mkdir -p ~/.config/keysync

# Generate completion script
keysync completion zsh > ~/.config/keysync/completion.zsh

# Add to ~/.zshrc:
echo 'source ~/.config/keysync/completion.zsh' >> ~/.zshrc
```

### fish

**Current session** (temporary):
```fish
keysync completion fish | source
```

**Persistent** (survives restarts):
```fish
# Add to ~/.config/fish/config.fish
echo 'keysync completion fish | source' >> ~/.config/fish/config.fish
```

**Alternative** (fish completions directory):
```fish
# Generate to fish completions directory (recommended)
keysync completion fish > ~/.config/fish/completions/keysync.fish
```

## Testing Completion

After setting up completion, test it:

1. Open a **new terminal session** (or run `source ~/.bashrc` / `source ~/.zshrc` / etc.)
2. Type `keysync s` and press **Tab** — should autocomplete to `keysync set`
3. Type `keysync set --p` and press **Tab** — should suggest `--project`
4. Type `keysync --store ` and press **Tab** — should suggest `fallback`

## Troubleshooting

### Completion doesn't work after setup

**Symptom**: Pressing Tab after `keysync` shows no suggestions.

**Solution**:
1. Make sure you opened a **new terminal session** after adding the completion line to your profile
2. Or reload your profile manually:
   - bash: `source ~/.bashrc`
   - zsh: `source ~/.zshrc`
   - fish: `source ~/.config/fish/config.fish`
   - PowerShell: `. $PROFILE`

### PowerShell says "File cannot be loaded because running scripts is disabled"

**Symptom**:
```
completion.ps1 cannot be loaded because running scripts is disabled on this system.
```

**Solution**: Change your PowerShell execution policy:
```powershell
# Check current policy
Get-ExecutionPolicy

# Allow local scripts (recommended)
Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
```

### bash: "command not found: _get_comp_words_by_ref"

**Symptom**: Error when sourcing bash completion.

**Solution**: Install `bash-completion` package:
```bash
# macOS (Homebrew)
brew install bash-completion@2

# Debian / Ubuntu
sudo apt-get install bash-completion

# Fedora
sudo dnf install bash-completion
```

Then add to `~/.bashrc`:
```bash
[[ -r "/usr/local/etc/profile.d/bash_completion.sh" ]] && . "/usr/local/etc/profile.d/bash_completion.sh"
```

### zsh: "command not found: compdef"

**Symptom**: Error when sourcing zsh completion.

**Solution**: Add this **before** the completion line in `~/.zshrc`:
```zsh
autoload -Uz compinit
compinit
```

### fish: "Unknown command: keysync"

**Symptom**: fish says keysync command doesn't exist when generating completion.

**Solution**: Make sure `keysync` is in your PATH before generating completion:
```fish
# Check if keysync is found
which keysync

# If not found, add to PATH first (example)
set -Ua fish_user_paths /usr/local/bin
```

## How It Works

Keysync uses [Cobra](https://github.com/spf13/cobra)'s built-in completion engine. When you press Tab:

1. Your shell calls the completion script
2. The script runs `keysync __complete` (a hidden command)
3. keysync returns suggestions based on the current context
4. Your shell displays the suggestions

This is **dynamic** — completion adapts to registered flags and commands, so as keysync adds new features, completion updates automatically (after you regenerate the script).

## Updating Completion

When you update keysync to a new version with new commands or flags:

1. Regenerate the completion script:
   ```bash
   keysync completion bash > ~/.config/keysync/completion.bash  # or zsh, fish, powershell
   ```
2. Reload your shell profile:
   ```bash
   source ~/.bashrc  # or ~/.zshrc, etc.
   ```

Or, if you use the inline method (e.g., `source <(keysync completion bash)`), just reload your profile — it regenerates on every shell startup automatically.

## Advanced: Custom Completions

Keysync registers custom completions for certain flags. For example:

- `--store` flag suggests `fallback` as a value
- Future releases may add completions for `--project` (suggesting project names from `.keysync.json`)

These are defined in `internal/commands/root.go` using cobra's `RegisterFlagCompletionFunc()`.

## Related Commands

- `keysync completion -h` — Show completion help
- `keysync completion bash -h` — Show bash-specific help
- `keysync completion zsh -h` — Show zsh-specific help
- `keysync completion fish -h` — Show fish-specific help
- `keysync completion powershell -h` — Show PowerShell-specific help

## Why Use Completion?

- **Speed**: Type less, autocomplete more
- **Accuracy**: Avoid typos in command and flag names
- **Discovery**: Press Tab to see available commands and flags without reading `--help`
- **Productivity**: Especially useful for long flag names like `--project` or `--env`

---

Return to [main README](../README.md) for full keysync documentation.
