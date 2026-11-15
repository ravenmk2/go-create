# go-create

[![Test](https://github.com/ravenmk2/go-create/actions/workflows/test.yml/badge.svg)](https://github.com/ravenmk2/go-create/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/ravenmk2/go-create)](https://github.com/ravenmk2/go-create/releases)
[![License](https://img.shields.io/github/license/ravenmk2/go-create)](LICENSE)

A Go project scaffolding tool: generates lib / cli / web project skeletons following established engineering conventions. It creates projects through a TUI wizard by default and also supports fully non-interactive creation with all parameters supplied. The template-driven design lets you add custom templates via template descriptors.

## Installation

```bash
go install github.com/ravenmk2/go-create/cmd/go-create@latest
```

## Usage

```bash
go-create                 # TUI wizard
go-create list            # concise template overview
go-create show web        # options, defaults, choices and examples
go-create new web myapi --module github.com/ravenmk2/myapi \
  --set http_framework=chi --set sqlite=true --yes
go-create new web myapi --module github.com/ravenmk2/myapi --yes --dry-run
                          # preview files and overwrites; no writes or post-actions
go-create upgrade         # self-update to the latest release (--check to only check, --yes to skip confirmation)
```

Custom templates live under `~/.config/go-create/templates/` on all platforms; a template with the same name takes precedence over the builtin one.

In the wizard, use Tab/Shift+Tab to move between fields, arrow keys to choose, and Space to toggle multiple selections. Enter advances or confirms. The final review lists files that would be overwritten; press E to jump to a section, edit it, and return directly to review. CLI-supplied name, module and license values can also be edited there. Overwrites require explicitly selecting **Overwrite & create**. Esc cancels; inside the section picker it returns to review.

`--dry-run` works with either the wizard or `--yes`. It validates and renders the same files as generation, including LICENSE, but does not create directories, modify files, initialize Git, or fetch dependencies. Preview reflects the filesystem at the time of inspection.
