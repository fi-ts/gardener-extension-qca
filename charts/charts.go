package charts

import (
	"embed"
)

// InternalChart embeds the cortex chart in embed.FS
//
//go:embed internal
var InternalChart embed.FS

const (
	// CortexCQCAChartsPathhartsPath is the path to the internal charts.
	QCAChartsPath = "qca"
	QCANamespace  = "kube-system"
	QCAName       = "qualys-cloud-agent"
)
