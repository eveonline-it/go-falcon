# Commit Guide - Enhanced with Changelog Integration

## How to Use Enhanced `/commit` Commands

When you want to commit changes **with automatic changelog updates**, use this workflow:

### 1. Standard Commit with Changelog Prompt

When you type `/commit`, I'll now ask you:

**"What type of change is this for the changelog?"**

Choose from:
- **`added`** - New features or functionality
- **`changed`** - Changes in existing functionality  
- **`fixed`** - Bug fixes and error corrections
- **`security`** - Security improvements
- **`deprecated`** - Soon-to-be removed features
- **`removed`** - Features that were completely removed

### 2. Example Workflow

```
You: /commit add user authentication system

Me: What type of change is this for the changelog?
1. added (new features)
2. changed (existing functionality changes)  
3. fixed (bug fixes)
4. security (security improvements)
5. deprecated (soon-to-be removed)
6. removed (removed features)
7. skip (don't update changelog)

Your choice (1-7):

You: 1

Me: [Creates commit + adds changelog entry automatically]
✅ Commit: "feat: add user authentication system"  
✅ Changelog: Added "User authentication system"
```

### 3. Skip Changelog Updates

If you want to commit without updating the changelog:

```
You: /commit fix typo in documentation --no-changelog

Me: [Creates commit without changelog prompt]
```

### 4. Batch Changelog Updates

For multiple small commits, you can update changelog separately:

```
You: /commit small fix
You: /commit another small fix  
You: /commit one more fix

Later...
You: Please add to changelog: "Multiple small fixes and improvements" as type "fixed"

Me: [Updates changelog with combined entry]
```

## Quick Reference

| Command | Action |
|---------|---------|
| `/commit message` | Normal commit with changelog prompt |
| `/commit message --no-changelog` | Skip changelog update |
| `/changelog-add fixed "Bug description"` | Add changelog entry only |
| `/changelog-show` | Show recent changelog entries |

## Conventional Commit Types → Changelog Types

| Commit Prefix | Changelog Type | Example |
|---------------|----------------|---------|
| `feat:` | `added` | New features |
| `fix:` | `fixed` | Bug fixes |
| `refactor:` | `changed` | Code improvements |
| `perf:` | `changed` | Performance improvements |
| `docs:` | `changed` | Documentation updates |
| `test:` | `changed` | Test additions |
| `chore:` | *(skip)* | Maintenance tasks |

## Benefits

✅ **Automatic changelog maintenance**  
✅ **Consistent formatting**  
✅ **No forgotten entries**  
✅ **Release-ready documentation**  
✅ **Better project history**