# <%if .Options.display_name%><% .Options.display_name %><%else%><% .Name %><%end%>

<%if .Repo%>[![Test](https://github.com/<% .Repo %>/actions/workflows/test.yml/badge.svg)](https://github.com/<% .Repo %>/actions/workflows/test.yml) <%if .Options.goreleaser%>[![Release](https://img.shields.io/github/v/release/<% .Repo %>)](https://github.com/<% .Repo %>/releases) <%end%><%if .License%>[![License](https://img.shields.io/github/license/<% .Repo %>)](LICENSE)<%end%>

<%end%><%if .Options.description%><% .Options.description %><%else%>One-line description: what this project is and what problem it solves.<%end%>
<%if or .Options.docker_image .Options.docker_dev%>

## Quick Start

<%if not .Repo%>Replace `<owner>/<repo>` with your GHCR repository path.

<%end%>```sh
docker run -d \
  --name <% .Name %> \
  --restart unless-stopped \
  -p 8080:8080 \
  ghcr.io/<%if .Repo%><% lower .Repo %><%else%><owner>/<repo><%end%>:<%if .Options.docker_image%>latest<%else%>dev<%end%>
```

Open http://localhost:8080 in your browser to access the web interface.
<%end%>
