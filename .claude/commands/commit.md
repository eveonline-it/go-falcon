Steps to follow strictly:

## Pre-Commit Checks
1. Run `git status` to understand all changes
2. Check for uncommitted CLAUDE-*.md memory bank files (exclude from commits)
3. If code changes exist:
   - Run tests for affected modules: `go test ./internal/[module]/...`
   - Run linting if available: `go fmt ./...` or configured linter
4. Review changes with `git diff` to understand impact

## Documentation Updates
5. For significant changes, update relevant documentation:
   - Module CLAUDE.md files if module behavior changed
   - Main CLAUDE.md if project-level changes
   - Update CLAUDE-activeContext.md with session progress (but don't commit)
   - Update CLAUDE-patterns.md if new patterns introduced (but don't commit)
   - Update CLAUDE-decisions.md if architectural decisions made (but don't commit)

## Staging Strategy
6. Stage changes selectively:
   - `git add -A` for all changes EXCEPT:
   - Never stage CLAUDE-activeContext.md (session-specific)
   - Never stage CLAUDE-temp.md (temporary scratch)
   - Carefully consider CLAUDE-patterns.md and CLAUDE-decisions.md (only if explicitly requested)
   - Use `git reset HEAD CLAUDE-*.md` to unstage memory bank files if needed

## Commit Message Generation
7. Generate conventional commit message following the pattern:
   ```
   <type>(<scope>): <subject>
   
   <body>
   
   <footer>
   ```
   
   Types:
   - feat: New feature
   - fix: Bug fix
   - docs: Documentation only
   - style: Code style (formatting, semicolons, etc)
   - refactor: Code refactoring
   - perf: Performance improvements
   - test: Adding tests
   - chore: Maintenance tasks
   - ci: CI/CD changes
   
   Scope: Module name (auth, scheduler, groups, etc.)
   
   Subject: Imperative mood, max 50 chars
   
   Body: Explain what and why, not how. Wrap at 72 chars.
   
   Footer: Breaking changes, issue references

## Final Steps
8. Display the commit for user review
9. Show files that were excluded (especially CLAUDE-*.md files)
10. Remind about push command if on feature branch
