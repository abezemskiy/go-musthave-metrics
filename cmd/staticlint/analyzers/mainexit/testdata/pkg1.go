package pkg1

import "os"

func main() {
	os.Exit(1) // want "using os.Exit in main function is prohibited"
}

func anotherFunc() {
	os.Exit(0) // No `want` comment here, so no report expected
}
