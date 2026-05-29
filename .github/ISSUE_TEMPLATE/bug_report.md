---
name: Bug report
about: Report a bug or unexpected behavior
title: ''
labels: bug
assignees: ''
---

## Description

A clear description of what the bug is.

## Steps to Reproduce

1. Run command: `keysync ...`
2. Expected behavior: ...
3. Actual behavior: ...

## Environment

- **OS**: (macOS, Linux, Windows)
- **OS Version**: (e.g., macOS 14.5, Ubuntu 24.04, Windows 11)
- **keysync version**: (run `keysync --version`)
- **Go version**: (run `go version`)
- **Client library** (if applicable): (Go, Python, TypeScript, Swift, Java, C#, Rust, C++, Ruby)

## Output

```
Paste relevant command output here
```

## Configuration

If relevant, share your `.keysync.json` (remove any sensitive project IDs):

```json
{
  "repos": {
    "yourorg/yourrepo": {
      "project": "yourproject"
    }
  }
}
```

## Additional Context

- Does this happen consistently or intermittently?
- Any error messages or warnings?
- Does the OS keychain work for other applications?
