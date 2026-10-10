# <%if .Options.display_name%><% .Options.display_name %><%else%><% .Name %><%end%>

<%if .Repo%>[![Test](https://github.com/<% .Repo %>/actions/workflows/test.yml/badge.svg)](https://github.com/<% .Repo %>/actions/workflows/test.yml)
<%if .Options.goreleaser%>[![Release](https://img.shields.io/github/v/release/<% .Repo %>)](https://github.com/<% .Repo %>/releases)
<%end%><%if .License%>[![License](https://img.shields.io/github/license/<% .Repo %>)](LICENSE)
<%end%><%end%>
<%if .Options.description%><% .Options.description %><%else%>One-line intro: what this project is and what problem it solves.<%end%>

## Installation

```sh
go install <% .Module %>/cmd/<% .Name %>@latest
```

## Build and Release

```sh
scripts/build.sh             # local multi-platform build into dist/
scripts/build.sh --install   # build only for the current platform and install to ~/.local/bin
```
<%if .Options.goreleaser%>
Releases are handled by goreleaser: pushing a `v*.*.*` tag triggers the Release workflow.
<%end%>
