package main

import (
	"fmt"
	"github.com/cloudfoundry/conda-cnb/conda"
	"os"
	"path/filepath"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/helper"
)

func main() {
	context, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create a default detection context: %s", err)
		os.Exit(100)
	}

	if err := context.BuildPlan.Init(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize Build Plan: %s\n", err)
		os.Exit(101)
	}

	code, err := runDetect(context)
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runDetect(context detect.Detect) (int, error) {
	if exists, err := helper.FileExists(filepath.Join(context.Application.Root, "environment.yml")); err != nil {
		return detect.FailStatusCode, err
	} else if !exists {
		context.Logger.Error("no environment.yml in app root")
		return context.Fail(), nil
	}

	dep, ok := context.BuildPlan[conda.PythonLayer]
	if !ok {
		return detect.FailStatusCode, fmt.Errorf("no python in buildplan")
	} else if dep.Version == "" {
		context.Logger.Error("no python version in buildplan")
		return detect.FailStatusCode, nil
	}

	return context.Pass(buildplan.BuildPlan{
		conda.Layer: buildplan.Dependency{
			Metadata: buildplan.Metadata{
				"build": true,
			},
		},
	})
}
