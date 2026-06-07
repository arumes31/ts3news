package bot

import (
	"math/rand"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	rand.Seed(1)
	os.Exit(m.Run())
}
