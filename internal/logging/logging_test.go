package logging

import "testing"

func TestInitVerbose(t *testing.T) {
	Init(true)
}

func TestInitQuiet(t *testing.T) {
	Init(false)
}
