package integration_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/conda-cnb/conda"
	"github.com/cloudfoundry/dagger"
	"gopkg.in/yaml.v2"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

var (
	bpDir, condaURI string
	err             error
)

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)
	bpDir, err = dagger.FindBPRoot()
	Expect(err).NotTo(HaveOccurred())
	condaURI, err = dagger.PackageBuildpack(bpDir)
	Expect(err).ToNot(HaveOccurred())
	defer os.RemoveAll(condaURI)

	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, _ spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	it("builds successfully and reuses the conda cache on a re-build with a simple conda app", func() {
		appRoot := filepath.Join("testdata", "simple_app")
		pythonVersion, err := readPythonVersion(filepath.Join(appRoot, conda.EnvironmentFile))
		Expect(err).NotTo(HaveOccurred())

		app, err := dagger.PackBuild(appRoot, condaURI)
		Expect(err).NotTo(HaveOccurred())
		defer app.Destroy()
		Expect(app.BuildLogs()).To(MatchRegexp("Conda Packages.*: Contributing to layer"))

		app, err = dagger.PackBuildNamedImage(app.ImageName, appRoot, condaURI)
		Expect(err).NotTo(HaveOccurred())
		Expect(app.BuildLogs()).NotTo(MatchRegexp("Conda Packages.*: Reusing cached layer"))
		Expect(app.BuildLogs()).NotTo(ContainSubstring("Downloading and Extracting Packages")) // Shows that conda is caching

		app.Env["PORT"] = "8080"
		Expect(app.Start()).To(Succeed())
		body, _, err := app.HTTPGet("/")
		Expect(err).ToNot(HaveOccurred())
		Expect(body).To(ContainSubstring("Hello, world!"))
		Expect(body).To(ContainSubstring("Using python: " + pythonVersion))
	})

	it("uses package-list.txt as a lockfile for re-builds", func() {
		appRoot := filepath.Join("testdata", "with_lock_file")
		app, err := dagger.PackBuild(appRoot, condaURI)
		Expect(err).NotTo(HaveOccurred())
		defer app.Destroy()

		app, err = dagger.PackBuildNamedImage(app.ImageName, appRoot, condaURI)
		Expect(err).NotTo(HaveOccurred())

		Expect(app.BuildLogs()).To(MatchRegexp("Conda Packages.*: Reusing cached layer"))

		app.Env["PORT"] = "8080"
		Expect(app.Start()).To(Succeed())
		body, _, err := app.HTTPGet("/")
		Expect(err).ToNot(HaveOccurred())
		Expect(body).To(ContainSubstring("Hello, world!"))
	})

	it("uses the vendored packages when the app is vendored", func() {
		app, err := dagger.PackBuild(filepath.Join("testdata", "vendored"), condaURI)
		Expect(err).NotTo(HaveOccurred())
		defer app.Destroy()

		Expect(app.BuildLogs()).To(ContainSubstring("file:///workspace/vendor"))

		app.Env["PORT"] = "8080"
		Expect(app.Start()).To(Succeed())
		body, _, err := app.HTTPGet("/")
		Expect(err).ToNot(HaveOccurred())
		Expect(body).To(ContainSubstring("Hello, world!"))
	})
}

type EnvironmentYML struct {
	Dependencies []string `yaml:"dependencies"`
}

func readPythonVersion(environmentPath string) (string, error) {
	file, err := ioutil.ReadFile(environmentPath)
	if err != nil {
		return "", err
	}

	environmentYML := EnvironmentYML{}
	err = yaml.Unmarshal(file, &environmentYML)
	if err != nil {
		return "", err
	}

	for _, item := range environmentYML.Dependencies {
		if strings.HasPrefix(item, "python") {
			splitString := strings.Split(item, "=")
			if len(splitString) == 2 {
				return splitString[1], nil
			}
		}
	}

	return "", nil
}
