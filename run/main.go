package main

import (
	"os"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/cargo"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/draft"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/packit/scribe"
	"github.com/paketo-community/miniconda"
)

func main() {
	logger := scribe.NewLogger(os.Stdout)
	clock := chronos.DefaultClock
	entryResolver := draft.NewPlanner()
	dependencyManager := postal.NewService(cargo.NewTransport())
	scriptRunner := miniconda.NewScriptRunner(pexec.NewExecutable("bash"))

	packit.Run(
		miniconda.Detect(),
		miniconda.Build(entryResolver, dependencyManager, scriptRunner, logger, clock),
	)
}
