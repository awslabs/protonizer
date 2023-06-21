package cmd

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	verbose = true
	os.Exit(m.Run())
}
