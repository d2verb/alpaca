# Git Workflow

## Commit Message Format (Gitmoji)

```
<emoji> <description>

<optional body>
```

### Common Emojis

| Emoji | Usage |
|-------|-------|
| ✨ | New feature |
| 🐛 | Bug fix |
| ♻️ | Refactor |
| 📝 | Documentation |
| ✅ | Add/update tests |
| 🔧 | Configuration |
| 🎨 | Code style/format |
| 🚀 | Performance |
| 🔥 | Remove code/files |
| 🏗️ | Architecture changes |
| 🎉 | Initial commit |
| ➕ | Add dependency |
| ➖ | Remove dependency |

### Examples

```
✨ Add alpaca run command
🐛 Fix preset loading when path contains spaces
♻️ Extract llama-server process management
📝 Document CLI commands
✅ Add tests for preset loader
```

## Branch Strategy (GitHub Flow)

1. `main` is always deployable
2. Create feature branch from `main`
3. Open PR for review
4. Merge after CI passes
5. Delete feature branch

### Branch Naming

```
feature/<description>   # New feature
fix/<description>       # Bug fix
docs/<description>      # Documentation
refactor/<description>  # Refactoring
```

Examples:
- `feature/add-pull-command`
- `fix/graceful-shutdown-timeout`
- `docs/cli-reference`

## Pull Request Workflow

When creating PRs:
1. Run `git diff main...HEAD` to see all changes
2. Analyze full commit history (not just latest commit)
3. Write comprehensive summary
4. Include test plan

## Do Not

- Do not force push to main
- Do not skip CI checks
- Do not commit secrets or credentials
- Do not commit large binary files
