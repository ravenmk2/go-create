# .goreleaser.yml
version: 2

builds:
  - id: <% .Name %>
    main: ./cmd/<% .Name %>
    binary: <% .Name %>
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    env: [CGO_ENABLED=0]
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}

checksum:
  name_template: checksums.txt

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^chore:"
      - "^ci:"
      - "^style:"
      - "^build:"
      - "Merge pull request"
  groups:
    - title: Features
      regexp: "^feat(\\(.+\\))?:.*$"
      order: 0
    - title: Bug Fixes
      regexp: "^fix(\\(.+\\))?:.*$"
      order: 1
<%if .Options.docker_image%>
dockers:
  - id: <% .Name %>-amd64
    goarch: amd64
    use: buildx
    dockerfile: Dockerfile
    image_templates:
      - ghcr.io/<% lower .Repo %>:{{ .Version }}-amd64
    build_flag_templates:
      - --platform=linux/amd64
      - --label=org.opencontainers.image.source=https://github.com/<% .Repo %>
  - id: <% .Name %>-arm64
    goarch: arm64
    use: buildx
    dockerfile: Dockerfile
    image_templates:
      - ghcr.io/<% lower .Repo %>:{{ .Version }}-arm64
    build_flag_templates:
      - --platform=linux/arm64
      - --label=org.opencontainers.image.source=https://github.com/<% .Repo %>

docker_manifests:
  - name_template: ghcr.io/<% lower .Repo %>:{{ .Version }}
    image_templates:
      - ghcr.io/<% lower .Repo %>:{{ .Version }}-amd64
      - ghcr.io/<% lower .Repo %>:{{ .Version }}-arm64
  - name_template: ghcr.io/<% lower .Repo %>:latest
    image_templates:
      - ghcr.io/<% lower .Repo %>:{{ .Version }}-amd64
      - ghcr.io/<% lower .Repo %>:{{ .Version }}-arm64
<%end%>
