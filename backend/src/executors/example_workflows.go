package executors

import (
	"os"
	"path/filepath"
	"runtime"
)

// ReadExampleWorkflow reads an example workflow file from the examples directory
func ReadExampleWorkflow(filename string) ([]byte, error) {
	// Get the current file's directory
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, nil
	}

	// Build path to examples directory: src/workflows/examples/
	baseDir := filepath.Dir(currentFile)       // src/executors
	srcDir := filepath.Dir(baseDir)            // src
	examplesDir := filepath.Join(srcDir, "workflows", "examples")
	filePath := filepath.Join(examplesDir, filename)

	return os.ReadFile(filePath)
}

