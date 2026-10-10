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
go-create list            # list available templates and their options
go-create new web myapi --module github.com/ravenmk2/myapi \
  --set http_framework=chi --set sqlite=true --yes
```

Custom templates live under `~/.config/go-create/templates/` on all platforms; a template with the same name takes precedence over the builtin one.
