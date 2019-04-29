package main

import (
	"fmt"
	"os"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/conda-cnb/conda"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/runner"
)

func main() {
	context, err := build.DefaultBuild()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create a default build context: %s", err)
		os.Exit(101)
	}

	code, err := runBuild(context)
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runBuild(context build.Build) (int, error) {
	context.Logger.FirstLine(context.Logger.PrettyIdentity(context.Buildpack))

	packagesContributor, willContribute, err := conda.NewContributor(context, runner.CommandRunner{})
	if err != nil {
		return context.Failure(102), err
	}

	if willContribute {
		if err := packagesContributor.Contribute(); err != nil {
			return context.Failure(103), err
		}
	}

	return context.Success(buildplan.BuildPlan{})
}
