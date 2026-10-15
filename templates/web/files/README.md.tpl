# <%if .Options.display_name%><% .Options.display_name %><%else%><% .Name %><%end%>

<%if .Repo%>[![Test](https://github.com/<% .Repo %>/actions/workflows/test.yml/badge.svg)](https://github.com/<% .Repo %>/actions/workflows/test.yml) <%if .Options.goreleaser%>[![Release](https://img.shields.io/github/v/release/<% .Repo %>)](https://github.com/<% .Repo %>/releases) <%end%><%if .License%>[![License](https://img.shields.io/github/license/<% .Repo %>)](LICENSE)<%end%>

<%end%><%if .Options.description%><% .Options.description %><%else%>One-line description: what this project is and what problem it solves.<%end%>
<%if .Options.docker_dev%>

Development image: pushes to `master`, `main`, `dev`, or `develop` (and manual
workflow runs) publish `ghcr.io/<owner>/<repo>:dev` after lint and tests pass.
Actions builds the frontend (when enabled) and a Linux amd64 Go binary, then
packages it with `Dockerfile`. The image path follows the GitHub repository,
lowercased. All branches share one concurrency group: the active run finishes
before another can publish; GitHub retains at most one pending run. This workflow
runs independently of `test.yml` and does not require GoReleaser.
<%end%>
