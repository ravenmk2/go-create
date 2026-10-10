# <%if .Options.display_name%><% .Options.display_name %><%else%><% .Name %><%end%>

[![Go Reference](https://pkg.go.dev/badge/<% .Module %>.svg)](https://pkg.go.dev/<% .Module %>)
[![Test](https://github.com/<% .Repo %>/actions/workflows/test.yml/badge.svg)](https://github.com/<% .Repo %>/actions/workflows/test.yml)
[![Release](https://img.shields.io/github/v/release/<% .Repo %>)](https://github.com/<% .Repo %>/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/<% .Repo %>)](go.mod)
<%if .License%>[![License](https://img.shields.io/github/license/<% .Repo %>)](LICENSE)
<%end%>
<%if .Options.description%><% .Options.description %><%else%>One-line introduction: what this project is and what problem it solves.<%end%>
