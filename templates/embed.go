package templates

import "embed"

// FS 内置模板集，每个一级子目录是一个模板（template.yaml + files/）。
//
//go:embed all:cli all:web all:lib
var FS embed.FS
