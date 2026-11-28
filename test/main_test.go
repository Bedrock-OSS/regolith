package test

import (
	"os"
	"testing"

	"github.com/Bedrock-OSS/regolith/regolith"
)

// TestMain initializes the logger, which the functions under test use and
// which is otherwise nil until InitLogging is called, so that every test of
// the package can run on its own.
func TestMain(m *testing.M) {
	regolith.InitLogging(true)
	os.Exit(m.Run())
}
