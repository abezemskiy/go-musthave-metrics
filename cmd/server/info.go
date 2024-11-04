package main

import (
	"fmt"
	"io"
)

// buildVersion - версия сборки.
var buildVersion string

// buildDate - дата сборки.
var buildDate string

// buildCommit - сообщение к сборке.
var buildCommit string

func printGlobalInfo(output io.Writer) {
	fmt.Fprint(output, "Build version: ")
	if buildVersion == "" {
		fmt.Fprint(output, "N/A")
	} else {
		fmt.Fprintf(output, "%s", buildVersion)
	}
	fmt.Fprint(output, "\n")

	fmt.Fprint(output, "Build date: ")
	if buildDate == "" {
		fmt.Fprint(output, "N/A")
	} else {
		fmt.Fprintf(output, "%s", buildDate)
	}
	fmt.Fprint(output, "\n")

	fmt.Fprint(output, "Build commit: ")
	if buildCommit == "" {
		fmt.Fprint(output, "N/A")
	} else {
		fmt.Fprintf(output, "%s", buildCommit)
	}
	fmt.Fprint(output, "\n")
}
