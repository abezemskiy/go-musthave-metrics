package main

import (
	"io"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/tools/info"
)

// buildVersion - версия сборки.
var buildVersion string

// buildDate - дата сборки.
var buildDate string

// buildCommit - сообщение к сборке.
var buildCommit string

func printGlobalInfo(output io.Writer) {
	info.Build(output, buildVersion, buildDate, buildCommit)
}
