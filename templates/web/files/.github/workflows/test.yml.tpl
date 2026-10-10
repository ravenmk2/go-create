# .github/workflows/test.yml
name: Test

on:
  push:
    branches: [main, master, develop, dev]
  pull_request:
  workflow_dispatch:

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7

      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
<%if .Options.embed_frontend%>
      - name: Build web frontend
        shell: bash
        run: bash scripts/build-web.sh
<%end%>
      - uses: golangci/golangci-lint-action@v9
        with:
          version: v2.13

  test:
    strategy:
      matrix:
        os:
<%range .Options.ci_platforms%>          - <% . %>
<%end%>    runs-on: ${{ matrix.os }}
    env:
      CGO_ENABLED: 0
    steps:
      - uses: actions/checkout@v7

      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
<%if .Options.embed_frontend%>
      - name: Build web frontend
        shell: bash
        run: bash scripts/build-web.sh
<%end%>
      - run: go vet ./...

      - run: go test ./...
