package conda_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/conda-cnb/conda"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -destination=mocks_test.go -package=conda_test github.com/cloudfoundry/libcfbuildpack/runner Runner

func TestUnitConda(t *testing.T) {
	spec.Run(t, "Conda", testConda, spec.Report(report.Terminal{}))
}

func testConda(t *testing.T, when spec.G, it spec.S) {
	var (
		f                    *test.BuildFactory
		mockCtrl             *gomock.Controller
		mockRunner           *MockRunner
		stubMinicondaFixture string
	)
	it.Before(func() {
		stubMinicondaFixture = filepath.Join("testdata", "stub-installer.sh")
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockRunner = NewMockRunner(mockCtrl)

		f = test.NewBuildFactory(t)
	})
	it.After(func() {
		mockCtrl.Finish()
	})

	when("NewContributor", func() {
		it("does not contribute when conda is not in the build plan", func() {
			_, willContribute, err := conda.NewContributor(f.Build, mockRunner)
			Expect(err).ToNot(HaveOccurred())
			Expect(willContribute).To(BeFalse())
		})

		it("does contribute when conda is in the buildplan", func() {
			f.AddBuildPlan(conda.CondaLayer, buildplan.Dependency{})

			_, willContribute, err := conda.NewContributor(f.Build, mockRunner)

			Expect(err).ToNot(HaveOccurred())
			Expect(willContribute).To(BeTrue())
		})
	})

	when("ContributeMiniconda", func() {
		it("installs miniconda", func() {
			f.AddBuildPlan(conda.CondaLayer, buildplan.Dependency{
				Version: "3",
				Metadata: buildplan.Metadata{
					"build":  true,
					"launch": true,
				},
			})

			minicondaLayerPath := filepath.Join(f.Build.Layers.Root, "miniconda3")

			f.AddDependency("miniconda3", stubMinicondaFixture)
			mockRunner.EXPECT().Run("stub-installer.sh", gomock.Any(), "-b", "-p", minicondaLayerPath)

			contributor, _, err := conda.NewContributor(f.Build, mockRunner)

			Expect(err).ToNot(HaveOccurred())
			Expect(contributor.ContributeMiniconda()).To(Succeed())

			minicondaLayer := f.Build.Layers.Layer("miniconda3")
			Expect(minicondaLayer).To(test.HaveLayerMetadata(false, true, false))
		})
	})

	when("ContributeCondaPackages", func() {
		var (
			condaPackagesLayer layers.Layer
			contributor        conda.Contributor
			err                error
		)

		it.Before(func() {
			f.AddBuildPlan(conda.CondaLayer, buildplan.Dependency{})
			contributor, _, err = conda.NewContributor(f.Build, mockRunner)
			Expect(err).NotTo(HaveOccurred())

			condaPackagesLayer = f.Build.Layers.Layer(conda.CondaPackagesLayer)
		})

		it.After(func() {
			Expect(condaPackagesLayer).To(test.HaveLayerMetadata(false, false, true))
		})

		when("a vendor dir does NOT exist", func() {
			var condaCacheLayer layers.Layer

			it.Before(func() {
				condaCacheLayer = f.Build.Layers.Layer(conda.CondaCacheLayer)
				mockRunner.EXPECT().Run(gomock.Any(), f.Build.Application.Root, "clean", "-pt")
			})

			it.After(func() {
				Expect(condaCacheLayer).To(test.HaveLayerMetadata(false, true, false))
				Expect(os.Getenv(conda.CondaPkgsDirs)).To(Equal(condaCacheLayer.Root))
			})

			when("a package-list.txt file does NOT exist", func() {
				it("builds", func() {
					mockRunner.EXPECT().Run(gomock.Any(), f.Build.Application.Root, "env", "update", "--prefix", condaPackagesLayer.Root, "--file", conda.EnvironmentFile)

					Expect(contributor.ContributeCondaPackages()).To(Succeed())
				})
			})

			when("a package-list.txt file exists", func() {
				it.Before(func() {
					writePackageListFile(f.Build.Application.Root)
				})

				it("builds", func() {
					mockRunner.EXPECT().Run(gomock.Any(), f.Build.Application.Root, "create", "--file", conda.LockFile, "--prefix", condaPackagesLayer.Root, "--yes", "--quiet")

					Expect(contributor.ContributeCondaPackages()).To(Succeed())
				})
			})
		})

		when("a vendor dir does exist", func() {
			var vendorDir string
			it.Before(func() {
				vendorDir = filepath.Join(f.Build.Application.Root, "vendor")
				Expect(os.MkdirAll(vendorDir, 0666)).To(Succeed())

				writePackageListFile(f.Build.Application.Root)
			})

			it("installs packages as offline using the vendor dir", func() {
				mockRunner.EXPECT().Run(gomock.Any(), f.Build.Application.Root, "create", "--file", conda.LockFile, "--prefix", condaPackagesLayer.Root, "--yes", "--quiet", "--channel", vendorDir, "--override-channels", "--offline")

				Expect(contributor.ContributeCondaPackages()).To(Succeed())
			})
		})
	})

	when("ContributeStartCommand", func() {
		it("adds the start command to the application metadata", func() {
			f.AddBuildPlan(conda.CondaLayer, buildplan.Dependency{})
			Expect(helper.WriteFile(filepath.Join(f.Build.Application.Root, "Procfile"), 0666, "web: python app.py")).To(Succeed())

			contributor, _, err := conda.NewContributor(f.Build, mockRunner)
			Expect(err).NotTo(HaveOccurred())

			Expect(contributor.ContributeStartCommand()).To(Succeed())
			Expect(f.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{Processes: []layers.Process{{"web", "python app.py"}}}))
		})
	})
}

func writePackageListFile(path string) {
	pkgListPath := filepath.Join(path, conda.LockFile)
	pkgListContents := "# Don't hash this line\nsome-dep=1.2.3"

	Expect(ioutil.WriteFile(pkgListPath, []byte(pkgListContents), os.ModePerm)).To(Succeed())
}
