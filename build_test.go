package miniconda_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/miniconda"
	"github.com/paketo-buildpacks/miniconda/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"

	//nolint Ignore SA1019, informed usage of deprecated package
	"github.com/paketo-buildpacks/packit/v2/paketosbom"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir string
		cnbDir    string

		buffer *bytes.Buffer

		dependencyManager *fakes.DependencyManager
		runner            *fakes.Runner
		sbomGenerator     *fakes.SBOMGenerator

		build        packit.BuildFunc
		buildContext packit.BuildContext
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:       "miniconda3",
			Name:     "miniconda3-dependency-name",
			Checksum: "miniconda3-dependency-sha",
			Stacks:   []string{"some-stack"},
			URI:      "miniconda3-dependency-uri",
			Version:  "miniconda3-dependency-version",
		}

		// Legacy SBOM
		dependencyManager.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "miniconda3",
				Metadata: paketosbom.BOMMetadata{
					Checksum: paketosbom.BOMChecksum{
						Algorithm: paketosbom.SHA256,
						Hash:      "miniconda3-dependency-sha",
					},
					URI:     "miniconda3-dependency-uri",
					Version: "miniconda3-dependency-version",
				},
			},
		}

		runner = &fakes.Runner{}

		// Syft SBOM
		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateFromDependencyCall.Returns.SBOM = sbom.SBOM{}

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		build = miniconda.Build(
			dependencyManager,
			runner,
			sbomGenerator,
			logEmitter,
			chronos.DefaultClock,
		)
		buildContext = packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				Version:     "some-version",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
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
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	it("returns a result that installs conda", func() {
		result, err := build(buildContext)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(1))
		layer := result.Layers[0]

		Expect(layer.Name).To(Equal("conda"))
		Expect(layer.Path).To(Equal(filepath.Join(layersDir, "conda")))

		Expect(layer.SharedEnv).To(BeEmpty())
		Expect(layer.BuildEnv).To(BeEmpty())
		Expect(layer.LaunchEnv).To(BeEmpty())
		Expect(layer.ProcessLaunchEnv).To(BeEmpty())

		Expect(layer.Build).To(BeFalse())
		Expect(layer.Launch).To(BeFalse())
		Expect(layer.Cache).To(BeFalse())

		Expect(layer.Metadata).To(HaveLen(1))
		Expect(layer.Metadata["dependency-sha"]).To(Equal("miniconda3-dependency-sha"))

		Expect(layer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
			{
				Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
			},
			{
				Extension: sbom.Format(sbom.SPDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
			},
		}))

		Expect(dependencyManager.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(dependencyManager.ResolveCall.Receives.Id).To(Equal("miniconda3"))
		Expect(dependencyManager.ResolveCall.Receives.Version).To(Equal("*"))
		Expect(dependencyManager.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
			{
				ID:       "miniconda3",
				Name:     "miniconda3-dependency-name",
				Checksum: "miniconda3-dependency-sha",
				Stacks:   []string{"some-stack"},
				URI:      "miniconda3-dependency-uri",
				Version:  "miniconda3-dependency-version",
			},
		}))

		Expect(dependencyManager.DeliverCall.Receives.Dependency).To(Equal(
			postal.Dependency{
				ID:       "miniconda3",
				Name:     "miniconda3-dependency-name",
				Checksum: "miniconda3-dependency-sha",
				Stacks:   []string{"some-stack"},
				URI:      "miniconda3-dependency-uri",
				Version:  "miniconda3-dependency-version",
			}))
		Expect(dependencyManager.DeliverCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.DeliverCall.Receives.DestinationPath).To(Equal(filepath.Join(layersDir, "miniconda-script-temp-layer")))
		Expect(dependencyManager.DeliverCall.Receives.PlatformPath).To(Equal("some-platform-path"))

		Expect(runner.RunCall.Receives.RunPath).To(Equal(filepath.Join(layersDir, "miniconda-script-temp-layer", "miniconda3-dependency-name")))
		Expect(runner.RunCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "conda")))

		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dir).To(Equal(filepath.Join(layersDir, "conda")))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
		Expect(buffer.String()).To(ContainSubstring("Installing Miniconda"))
	})

	context("when the conda layer is required at build and launch", func() {
		it.Before(func() {
			buildContext.Plan.Entries[0].Metadata = make(map[string]interface{})
			buildContext.Plan.Entries[0].Metadata["launch"] = true
			buildContext.Plan.Entries[0].Metadata["build"] = true
		})

		it("returns a layer with build and launch set true and the BOM is set for build and launch", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			layer := result.Layers[0]

			Expect(layer.Name).To(Equal("conda"))

			Expect(layer.Build).To(BeTrue())
			Expect(layer.Launch).To(BeTrue())
			Expect(layer.Cache).To(BeTrue())

			Expect(result.Build.BOM).To(Equal(
				[]packit.BOMEntry{
					{
						Name: "miniconda3",
						Metadata: paketosbom.BOMMetadata{
							Checksum: paketosbom.BOMChecksum{
								Algorithm: paketosbom.SHA256,
								Hash:      "miniconda3-dependency-sha",
							},
							URI:     "miniconda3-dependency-uri",
							Version: "miniconda3-dependency-version",
						},
					},
				},
			))

			Expect(result.Launch.BOM).To(Equal(
				[]packit.BOMEntry{
					{
						Name: "miniconda3",
						Metadata: paketosbom.BOMMetadata{
							Checksum: paketosbom.BOMChecksum{
								Algorithm: paketosbom.SHA256,
								Hash:      "miniconda3-dependency-sha",
							},
							URI:     "miniconda3-dependency-uri",
							Version: "miniconda3-dependency-version",
						},
					},
				},
			))
		})
	})

	context("failure cases", func() {
		context("when the dependency manager resolution fails", func() {
			it.Before(func() {
				dependencyManager.ResolveCall.Returns.Error = errors.New("resolve call failed")
			})

			it("returns an error", func() {
				_, err := build(buildContext)

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
				_, err := build(buildContext)

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
				_, err := build(buildContext)

				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the dependency manager delivery fails", func() {
			it.Before(func() {
				dependencyManager.DeliverCall.Returns.Error = errors.New("deliver call failed")
			})

			it("returns an error", func() {
				_, err := build(buildContext)

				Expect(err).To(MatchError("deliver call failed"))
			})
		})

		context("when the dependency manager resolution fails", func() {
			it.Before(func() {
				runner.RunCall.Returns.Error = errors.New("run call failed")
			})

			it("returns an error", func() {
				_, err := build(buildContext)

				Expect(err).To(MatchError("run call failed"))
			})
		})

		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				buildContext.BuildpackInfo.SBOMFormats = []string{"random-format"}
			})

			it("returns an error", func() {
				_, err := build(buildContext)

				Expect(err).To(MatchError(`unsupported SBOM format: 'random-format'`))
			})
		})

		context("when formatting the SBOM returns an error", func() {
			it.Before(func() {
				sbomGenerator.GenerateFromDependencyCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(buildContext)

				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})
	})
}
