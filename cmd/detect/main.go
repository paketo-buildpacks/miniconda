package main

import (
	"fmt"
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/conda-cnb/conda"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"os"
	"path/filepath"
)

func main() {
	context, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create a default detection context: %s", err)
		os.Exit(100)
	}

	code, err := runDetect(context)
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runDetect(context detect.Detect) (int, error) {
	environmentYMLPath := filepath.Join(context.Application.Root, conda.EnvironmentFile)
	if exists, err := helper.FileExists(environmentYMLPath); err != nil {
		return detect.FailStatusCode, err
	} else if !exists {
		return context.Fail(), nil
	}

	return context.Pass(buildplan.Plan{
		Requires: []buildplan.Required {
			{
				Name: conda.CondaLayer,
				Metadata: buildplan.Metadata{
					"build":  true,
					"launch": true,
				},
			},

		},
		Provides: []buildplan.Provided {
			{Name: conda.CondaLayer},
		},
	})
}
