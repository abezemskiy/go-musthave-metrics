package info

import (
	"fmt"
	"io"
)

func Build(output io.Writer, buildVersion, buildDate, buildCommit string) {
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
