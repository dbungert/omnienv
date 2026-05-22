# Conventional Commits Cheatsheet

```
<type>(<scope>): <description>

<body>

<footer>
```

## Types

| Type       | Usage                                  |
|------------|----------------------------------------|
| `feat`     | A new feature                          |
| `fix`      | A bug fix                              |
| `build`    | Build system or dependencies           |
| `chore`    | Maintenance, config, tooling           |
| `ci`       | CI config or scripts                   |
| `docs`     | Documentation only                     |
| `perf`     | Performance improvement                |
| `refactor` | Code change with no behavior change    |
| `revert`   | Revert a prior commit                  |
| `style`    | Formatting, whitespace, lint           |
| `test`     | Adding or fixing tests                 |

## Scopes (project-specific)

| Scope       | Area                                     |
|-------------|------------------------------------------|
| `launch`    | Environment creation / LXD provisioning  |
| `shell`     | Shell session / lxc exec                 |
| `cfg`       | Config file parsing, `.omnienv.yaml`     |
| `opts`      | CLI option parsing                       |
| `user`      | User/UID mapping                         |
| `log`       | Logging setup                            |
| `doc`       | Documentation                            |
| `build`     | Makefile, pre-commit, go.mod             |

## Examples

```
feat(launch): add VM support via --vm flag

fix(cfg): handle empty map in UnmarshalYAML without panic

refactor: extract LXD client into interface

docs: add Quick Start section to README

test(opts): cover PassAfterNonOption edge case

build: bump LXD client dependency to v0.0.0-20250314

fix: typo in error message for cloud-init timeout
```

## Breaking Changes

Append `!` after the type/scope, or add `BREAKING CHANGE:` in the footer:

```
feat(launch)!: remove --system override flag

fix(cfg)!: rename 'series' key to 'system'
BREAKING CHANGE: The 'series' config key is no longer accepted.
```

## Rules

- Description: imperative, lowercase, no trailing period
- Scope: optional but encouraged; use a single word
- Body: optional, wrap at 72 chars, explain *why* not *what*
- Footer: optional, for breaking changes or issue references
