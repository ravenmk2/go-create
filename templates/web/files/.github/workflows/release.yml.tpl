# .github/workflows/release.yml
name: Release

on:
  push:
    tags: ["v*.*.*"]
  workflow_dispatch:

permissions:
  contents: write
<%if .Options.docker_image%>  packages: write   # push to GHCR
<%end%>
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
<%if .Options.docker_image%>
      - uses: docker/setup-qemu-action@v4       # register binfmt for arm64 image builds
      - uses: docker/setup-buildx-action@v4     # prerequisite for use: buildx in dockers

      - uses: docker/login-action@v4
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
<%end%>
      - uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
