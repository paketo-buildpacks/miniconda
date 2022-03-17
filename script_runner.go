package miniconda

import (
	"fmt"
	"os"

	"github.com/paketo-buildpacks/packit/v2/pexec"
)

//go:generate faux --interface Executable --output fakes/executable.go

// Executable defines the interface for invoking an executable.
type Executable interface {
	Execute(execution pexec.Execution) error
}

// ScriptRunner implements the Runner interface
type ScriptRunner struct {
	executable Executable
}

// NewScriptRunner creates an instance of the ScriptRunner given an Executable that runs `bash`.
func NewScriptRunner(executable Executable) ScriptRunner {
	return ScriptRunner{
		executable: executable,
	}
}

// Run invokes the miniconda script located in the given runPath, which
// installs conda into the a layer path designated by condaLayerPath.
func (s ScriptRunner) Run(runPath, condaLayerPath string) error {

	err := os.Chmod(runPath, 0550)
	if err != nil {
		return err
	}

	err = s.executable.Execute(pexec.Execution{
		Args: []string{
			runPath,
			"-b",
			"-f",
			"-p", condaLayerPath,
		},
	})
	if err != nil {
		return fmt.Errorf("failed while running miniconda install script: %w", err)
	}

	return nil
}
