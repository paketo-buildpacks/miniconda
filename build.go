package miniconda

import (
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
//go:generate faux --interface Runner --output fakes/runner.go

// DependencyManager defines the interface for picking the best matching
// dependency and installing it.
type DependencyManager interface {
	Resolve(path, id, version, stack string) (postal.Dependency, error)
	Deliver(dependency postal.Dependency, cnbPath, destinationPath, platformPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

// EntryResolver defines the interface for picking the most relevant entry from
// the Buildpack Plan entries.
type EntryResolver interface {
	MergeLayerTypes(name string, entries []packit.BuildpackPlanEntry) (launch, build bool)
}

// Runner defines the interface for invoking the miniconda script downloaded as a dependency.
type Runner interface {
	Run(runPath, layerPath string) error
}

// Build will return a packit.BuildFunc that will be invoked during the build
// phase of the buildpack lifecycle.
//
// Build will find the right miniconda dependency to download, download it
// into a layer, run the miniconda script to install conda into a separate
// layer and generate Bill-of-Materials. It also makes use of the checksum of
// the dependency to reuse the layer when possible.
func Build(entryResolver EntryResolver, dependencyManager DependencyManager, runner Runner, logger scribe.Logger, clock chronos.Clock) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		dependency, err := dependencyManager.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), "miniconda3", "*", context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		var bom = dependencyManager.GenerateBillOfMaterials(dependency)

		condaLayer, err := context.Layers.Get("conda")
		if err != nil {
			return packit.BuildResult{}, err
		}

		var buildMetadata = packit.BuildMetadata{}
		var launchMetadata = packit.LaunchMetadata{}

		cachedSHA, ok := condaLayer.Metadata[DepKey].(string)
		if ok && cachedSHA == dependency.SHA256 {

			logger.Process("Reusing cached layer %s", condaLayer.Path)
			logger.Break()

			if condaLayer.Build {
				buildMetadata = packit.BuildMetadata{BOM: bom}
			}

			if condaLayer.Launch {
				launchMetadata = packit.LaunchMetadata{BOM: bom}
			}

			return packit.BuildResult{
				Layers: []packit.Layer{condaLayer},
				Build:  buildMetadata,
				Launch: launchMetadata,
			}, nil
		}

		condaLayer, err = condaLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		condaLayer.Launch, condaLayer.Build = entryResolver.MergeLayerTypes("conda", context.Plan.Entries)
		condaLayer.Cache = condaLayer.Build

		minicondaScriptTempLayer, err := context.Layers.Get("miniconda-script-temp-layer")
		if err != nil {
			return packit.BuildResult{}, err
		}

		minicondaScriptTempLayer, err = minicondaScriptTempLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Process("Executing build process")
		logger.Subprocess("Installing Miniconda %s", dependency.Version)

		duration, err := clock.Measure(func() error {
			return dependencyManager.Deliver(dependency, context.CNBPath, minicondaScriptTempLayer.Path, context.Platform.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		condaLayer.Metadata = map[string]interface{}{
			DepKey:     dependency.SHA256,
			"built_at": clock.Now().Format(time.RFC3339Nano),
		}

		scriptPath := filepath.Join(minicondaScriptTempLayer.Path, "artifact")
		err = runner.Run(scriptPath, condaLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		if condaLayer.Build {
			buildMetadata = packit.BuildMetadata{BOM: bom}
		}

		if condaLayer.Launch {
			launchMetadata = packit.LaunchMetadata{BOM: bom}
		}

		return packit.BuildResult{
			Layers: []packit.Layer{condaLayer},
			Build:  buildMetadata,
			Launch: launchMetadata,
		}, nil
	}
}
