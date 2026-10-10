# <%if .Options.display_name%><% .Options.display_name %><%else%><% .Name %><%end%>

<%if .Repo%>[![Test](https://github.com/<% .Repo %>/actions/workflows/test.yml/badge.svg)](https://github.com/<% .Repo %>/actions/workflows/test.yml) <%if .Options.goreleaser%>[![Release](https://img.shields.io/github/v/release/<% .Repo %>)](https://github.com/<% .Repo %>/releases) <%end%><%if .License%>[![License](https://img.shields.io/github/license/<% .Repo %>)](LICENSE)<%end%>

<%end%><%if .Options.description%><% .Options.description %><%else%>One-line description: what this project is and what problem it solves.<%end%>
