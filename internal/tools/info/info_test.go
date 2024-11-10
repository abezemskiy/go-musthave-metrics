package info

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuild(t *testing.T) {

	{
		var output bytes.Buffer
		buildVersion := ""
		buildDate := ""
		buildCommit := ""
		Build(&output, buildVersion, buildDate, buildCommit)
		wantResult := "Build version: N/A\nBuild date: N/A\nBuild commit: N/A\n"
		assert.Equal(t, wantResult, output.String())
	}
	{
		var output bytes.Buffer
		buildVersion := "1.1"
		buildDate := "05.11.2024 9:32"
		buildCommit := "Hello world"
		Build(&output, buildVersion, buildDate, buildCommit)
		wantResult := "Build version: 1.1\nBuild date: 05.11.2024 9:32\nBuild commit: Hello world\n"
		assert.Equal(t, wantResult, output.String())
	}
}
