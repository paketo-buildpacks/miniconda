package miniconda_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/packit/scribe"
	"github.com/paketo-community/miniconda"
	"github.com/paketo-community/miniconda/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir string
		cnbDir    string

		buffer    *bytes.Buffer
		timeStamp time.Time
		clock     chronos.Clock

		dependencyManager *fakes.DependencyManager
		runner            *fakes.Runner
		logger            scribe.Logger
		entryResolver     *fakes.EntryResolver

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:      "miniconda3",
			Name:    "miniconda3-dependency-name",
			SHA256:  "miniconda3-dependency-sha",
			Stacks:  []string{"some-stack"},
			URI:     "miniconda3-dependency-uri",
			Version: "miniconda3-dependency-version",
		}

		dependencyManager.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "miniconda3",
				Metadata: map[string]interface{}{
					"version": "miniconda3-dependency-version",
					"name":    "miniconda3-dependency-name",
					"sha256":  "miniconda3-dependency-sha",
					"stacks":  []string{"some-stack"},
					"uri":     "miniconda3-dependency-uri",
				},
			},
		}

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return timeStamp
		})

		runner = &fakes.Runner{}

		entryResolver = &fakes.EntryResolver{}

		buffer = bytes.NewBuffer(nil)
		logger = scribe.NewLogger(buffer)

		build = miniconda.Build(entryResolver, dependencyManager, runner, logger, clock)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	it("returns a result that installs conda", func() {
		result, err := build(packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
			CNBPath: cnbDir,
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{Name: "conda"},
				},
			},
			Platform: packit.Platform{Path: "some-platform-path"},
			Layers:   packit.Layers{Path: layersDir},
			Stack:    "some-stack",
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result).To(Equal(packit.BuildResult{
			Layers: []packit.Layer{
				{
					Name:             "conda",
					Path:             filepath.Join(layersDir, "conda"),
					SharedEnv:        packit.Environment{},
					BuildEnv:         packit.Environment{},
					LaunchEnv:        packit.Environment{},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            false,
					Launch:           false,
					Cache:            false,
					Metadata: map[string]interface{}{
						miniconda.DepKey: "miniconda3-dependency-sha",
						"built_at":       timeStamp.Format(time.RFC3339Nano),
					},
				},
			},
		}))

		Expect(dependencyManager.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(dependencyManager.ResolveCall.Receives.Id).To(Equal("miniconda3"))
		Expect(dependencyManager.ResolveCall.Receives.Version).To(Equal("*"))
		Expect(dependencyManager.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
			{
				ID:      "miniconda3",
				Name:    "miniconda3-dependency-name",
				SHA256:  "miniconda3-dependency-sha",
				Stacks:  []string{"some-stack"},
				URI:     "miniconda3-dependency-uri",
				Version: "miniconda3-dependency-version",
			},
		}))

		Expect(entryResolver.MergeLayerTypesCall.Receives.Name).To(Equal("conda"))
		Expect(entryResolver.MergeLayerTypesCall.Receives.Entries).To(Equal([]packit.BuildpackPlanEntry{
			{Name: "conda"},
		}))

		Expect(dependencyManager.DeliverCall.Receives.Dependency).To(Equal(
			postal.Dependency{
				ID:      "miniconda3",
				Name:    "miniconda3-dependency-name",
				SHA256:  "miniconda3-dependency-sha",
				Stacks:  []string{"some-stack"},
				URI:     "miniconda3-dependency-uri",
				Version: "miniconda3-dependency-version",
			}))
		Expect(dependencyManager.DeliverCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.DeliverCall.Receives.DestinationPath).To(Equal(filepath.Join(layersDir, "miniconda-script-temp-layer")))
		Expect(dependencyManager.DeliverCall.Receives.PlatformPath).To(Equal("some-platform-path"))

		Expect(runner.RunCall.Receives.RunPath).To(Equal(filepath.Join(layersDir, "miniconda-script-temp-layer", "miniconda3-dependency-uri")))
		Expect(runner.RunCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "conda")))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
		Expect(buffer.String()).To(ContainSubstring("Installing Miniconda"))
	})

	context("when the conda layer is required at build and launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("returns a layer with build and launch set true and the BOM is set for build and launch", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				CNBPath: cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "conda"},
					},
				},
				Platform: packit.Platform{Path: "some-platform-path"},
				Layers:   packit.Layers{Path: layersDir},
				Stack:    "some-stack",
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:             "conda",
						Path:             filepath.Join(layersDir, "conda"),
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Launch:           true,
						Cache:            true,
						Metadata: map[string]interface{}{
							miniconda.DepKey: "miniconda3-dependency-sha",
							"built_at":       timeStamp.Format(time.RFC3339Nano),
						},
					},
				},
				Launch: packit.LaunchMetadata{
					BOM: []packit.BOMEntry{
						{
							Name: "miniconda3",
							Metadata: map[string]interface{}{
								"version": "miniconda3-dependency-version",
								"name":    "miniconda3-dependency-name",
								"sha256":  "miniconda3-dependency-sha",
								"stacks":  []string{"some-stack"},
								"uri":     "miniconda3-dependency-uri",
							},
						},
					},
				},
				Build: packit.BuildMetadata{
					BOM: []packit.BOMEntry{
						{
							Name: "miniconda3",
							Metadata: map[string]interface{}{
								"version": "miniconda3-dependency-version",
								"name":    "miniconda3-dependency-name",
								"sha256":  "miniconda3-dependency-sha",
								"stacks":  []string{"some-stack"},
								"uri":     "miniconda3-dependency-uri",
							},
						},
					},
				},
			}))
		})
	})

	context("failure cases", func() {
		context("when the dependency manager resolution fails", func() {
			it.Before(func() {
				dependencyManager.ResolveCall.Returns.Error = errors.New("resolve call failed")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError("resolve call failed"))
			})
		})

		context("when the layer dir cannot be accessed", func() {
			it.Before(func() {
				Expect(os.Chmod(layersDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Layers:  packit.Layers{Path: layersDir},
				})

				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the layer dir cannot be reset", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(layersDir, "conda", "bin"), os.ModePerm)).To(Succeed())
				Expect(os.Chmod(filepath.Join(layersDir, "conda"), 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, "conda"), os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Layers:  packit.Layers{Path: layersDir},
				})

				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the dependency manager delivery fails", func() {
			it.Before(func() {
				dependencyManager.DeliverCall.Returns.Error = errors.New("deliver call failed")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					Layers: packit.Layers{Path: layersDir},
				})

				Expect(err).To(MatchError("deliver call failed"))
			})
		})

		context("when the dependency manager resolution fails", func() {
			it.Before(func() {
				runner.RunCall.Returns.Error = errors.New("run call failed")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					Layers: packit.Layers{Path: layersDir},
				})

				Expect(err).To(MatchError("run call failed"))
			})
		})
	})
}
