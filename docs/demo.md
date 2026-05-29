# Terminal demo (asciinema)

Record **two** short casts if you can — install/setup vs day-to-day features:

| Recording | Suggested filename | Guide |
|-----------|-------------------|--------|
| Fresh install (Homebrew, `gh`, `keysync`) | `keysync-install.cast` | [homebrew.md](homebrew.md#fresh-machine-demo-script-asciinema) |
| Features (`set`, `push`, `export`, `mv`) | `keysync-features.cast` | Flow below |

A short terminal recording dramatically improves first impressions on the README. Recommended **features** flow (~30 seconds):

```bash
asciinema rec keysync-demo.cast
# In the recording:
keysync init --project my-app
keysync set API_KEY=demo_value_123
keysync list
keysync export --project my-app | head -3
# Ctrl-D to end, then:
asciinema upload keysync-demo.cast
```

Paste the embed snippet into the README **Demo** section:

```markdown
[![asciinema demo](https://asciinema.org/a/XXXXXX.svg)](https://asciinema.org/a/XXXXXX)
```

Replace `XXXXXX` with the recording ID from upload.

**Tips:**

- Use a clean terminal theme and large font
- Do not paste real secret values — use obvious placeholders
- Run `keysync doctor` at the end to show a healthy setup
