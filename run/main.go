package main

import (
	"os"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/miniconda"
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
