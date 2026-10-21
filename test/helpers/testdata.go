package helpers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// OriginalShow returns the factory show data of the iCubeSmart 3D8RGB cube.
func OriginalShow(test *testing.T) string {
	test.Helper()
	_, file, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "testdata", "original.bin"))
	if err != nil {
		test.Fatal(err)
	}

	return string(data)
}
