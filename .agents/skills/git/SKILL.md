---
name: git
description: Use when user needs git operations. Handles git commands, status checks, branching, committing, merging, and repository management. Covers git config, log inspection, diff analysis, stash management, and all core git workflows. Use when git-related tasks are requested (clone, init, add, commit, push, pull, merge, rebase, tag).
license: MIT
compatibility: "*"
metadata: {}
---

# Git Skill

This skill handles all git operations and workflows. It provides comprehensive git repository management capabilities including status checking, branching, staging, committing, remote operations, and repository maintenance.

## Core Capabilities

### Repository Operations
- Initialize new git repositories
- Clone existing repositories
- Check repository status and information
- Display git configuration
- View logs and history

### File Management
- Stage and unstage files
- Commit changes with proper formatting
- View staged and unstaged changes
- Revert and reset operations
- Stash management

### Branch Operations
- List and create branches
- Switch between branches
- Merge branches
- Rebase operations
- Branch deletion

### Remote Operations
- Add and remove remotes
- Fetch, pull, and push
- Manage remote tracking branches
- Repository synchronization

### Inspection and Analysis
- View diffs (changes)
- Show log history
- List untracked files
- Check repository health
- Analyze repository statistics

## Usage Examples

**Check repository status:**
```
Use git skill to check current repository status, staged and unstaged changes.
```

**Create and commit changes:**
```
Use git skill to add files, commit with appropriate message, and push to remote.
```

**Branch management:**
```
Use git skill to create feature branch, make changes, merge back to main.
```

**Resolve conflicts:**
```
Use git skill to view conflicted files, resolve issues, complete merge/rebase.
```

## Technical Details

This skill uses the local git command-line interface to perform operations. It handles:

- Git command execution and error handling
- Standard git workflows and conventions
- Branch and repository state management
- Interactive git operations

The skill supports both simple and complex git operations, making it suitable for day-to-day git usage as well as repository maintenance tasks.

## Integration

The git skill integrates with other tools for:

- File content reading/editing alongside git operations
- Project documentation generation
- Workflow automation and CI/CD support
- Version control verification

## Restrictions

- Operations are limited to local and remote repositories accessible via git protocol
- Branch and tag names follow git naming conventions
- No external git hosting platform integration beyond standard git commands
- Repository permissions depend on the underlying filesystem and git configuration