package conda_test

import (
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/conda-cnb/cmd/conda"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
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

	when("modules.NewContributor", func() {
		it("does not contribute when conda is not in the build plan", func() {
			_, willContribute, err := conda.NewContributor(f.Build, mockRunner)
			Expect(err).ToNot(HaveOccurred())
			Expect(willContribute).To(BeFalse())
		})

		it("does contribute when conda is in the buildplan", func() {
			f.AddBuildPlan(conda.MinicondaLayer, buildplan.Dependency{})

			_, willContribute, err := conda.NewContributor(f.Build, mockRunner)

			Expect(err).ToNot(HaveOccurred())
			Expect(willContribute).To(BeTrue())
		})
	})

	when("ContributeMiniconda", func() {
		it("installs the miniconda and sets PYTHONHOME", func() {
			f.AddBuildPlan(conda.MinicondaLayer, buildplan.Dependency{
				Version: "3",
				Metadata: buildplan.Metadata{
					"build":  true,
					"launch": true,
				},
			})

			minicondaLayerPath := filepath.Join(f.Build.Layers.Root, "miniconda3")

			f.AddDependency("miniconda3", stubMinicondaFixture)
			mockRunner.EXPECT().Run("./conda_installer.sh", gomock.Any(), "-b", "-p", minicondaLayerPath)

			contributor, _, err := conda.NewContributor(f.Build, mockRunner)

			Expect(err).ToNot(HaveOccurred())
			Expect(contributor.ContributeMiniconda()).To(Succeed())

			minicondaLayer := f.Build.Layers.Layer("miniconda3")
			Expect(minicondaLayer).To(test.HaveLayerMetadata(true, false, true))
		})
	})

	when("ContributeCondaModules", func() {
		it("creates conda modules layer", func() {
			f.AddBuildPlan(conda.MinicondaLayer, buildplan.Dependency{})

			contributor, _, err := conda.NewContributor(f.Build, mockRunner)
			Expect(err).NotTo(HaveOccurred())

			mockRunner.EXPECT().Run("conda", f.Build.Application.Root, gomock.Any())

			Expect(contributor.ContributeCondaModules("")).To(Succeed())

			condaModulesLayer := f.Build.Layers.Layer(conda.CondaModulesLayer)
			Expect(condaModulesLayer).To(test.HaveLayerMetadata(true, false, true))
			Expect(condaModulesLayer).To(test.HaveOverrideSharedEnvironment("PYTHONHOME", condaModulesLayer.Root))

		})
	})

	when("ContributeCondaModules", func() {

		it.Before(func() {
			appEnvYML := filepath.Join(f.Build.Application.Root, "environment.yml")
			Expect(helper.WriteFile(appEnvYML, os.ModePerm, `name: test_env
dependencies:
- python=2.7.15
- pandas=0.18.0`)).To(Succeed())
		})

		it.Focus("makes a new environment.yml with the python version from the buildplan", func() {
			f.AddBuildPlan(conda.PythonLayer, buildplan.Dependency{
				Version: "2.7.15",
				Metadata: buildplan.Metadata{
					"build":  true,
					"launch": true,
				},
			})

			// Need conda layer so stuff runs
			f.AddBuildPlan(conda.MinicondaLayer, buildplan.Dependency{
				Version: "2.6.7",
			})



			contributor, _, err := conda.NewContributor(f.Build, mockRunner)
			Expect(err).NotTo(HaveOccurred())

			//mockRunner.EXPECT().RunWithOutput("python", f.Build.Application.Root, "--version")

			mockRunner.EXPECT().Run("conda", f.Build.Application.Root, gomock.Any())

			Expect(contributor.ContributeCondaModules("2.7.15")).To(Succeed())

			path := f.Build.Layers.Layer(conda.EnvironmentYMLLayer)


			condaModulesLayer := f.Build.Layers.Layer(conda.CondaModulesLayer)
			Expect(condaModulesLayer).To(test.HaveLayerMetadata(true, false, true))
			Expect(condaModulesLayer).To(test.HaveOverrideSharedEnvironment("PYTHONHOME", condaModulesLayer.Root))
			files, err := ioutil.ReadDir(path.Root)
			Expect(err).ToNot(HaveOccurred())
			Expect(files).ToNot(BeEmpty())

			fileContents, err := ioutil.ReadFile(filepath.Join(path.Root, "modified_environment.yml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(fileContents)).To(ContainSubstring("python=2.7.15"))

			// No trampling
			Expect(string(fileContents)).To(ContainSubstring("pandas"))
		})
	})

}
