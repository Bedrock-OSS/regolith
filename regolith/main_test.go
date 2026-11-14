package regolith

import (
	"os"
	"testing"
)

// TestMain initializes the logger, which the functions under test use and
// which is otherwise nil until InitLogging is called, so that every test of
// the package can run on its own.
func TestMain(m *testing.M) {
	InitLogging(false)
	os.Exit(m.Run())
}
