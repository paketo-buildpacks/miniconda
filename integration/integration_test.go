package integration_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/cloudfoundry/conda-cnb/conda"
	"github.com/cloudfoundry/dagger"
	"gopkg.in/yaml.v2"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestIntegration(t *testing.T) {
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		condaURI string
		err      error
	)

	it.Before(func() {
		RegisterTestingT(t)

		Expect(err).ToNot(HaveOccurred())

		condaURI, err = dagger.PackageBuildpack()
		Expect(err).ToNot(HaveOccurred())
	})

	it.After(func() {
		os.RemoveAll(condaURI)
	})

	when("pushing a simple conda app", func() {
		var (
			app     *dagger.App
			appRoot string
			err     error
		)

		it.Before(func() {
			appRoot = filepath.Join("testdata", "simple_app")
		})

		it.After(func() {
			app.Destroy()
		})

		it("builds and runs", func() {
			app, err = dagger.PackBuild(appRoot, condaURI)
			Expect(err).NotTo(HaveOccurred())

			app.Env["PORT"] = "8080"
			Expect(app.Start()).To(Succeed())

			pythonVersion, err := readPythonVersion(filepath.Join(appRoot, conda.EnvironmentFile))
			Expect(err).NotTo(HaveOccurred())

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, world!"))
			Expect(body).To(ContainSubstring("Using python: " + pythonVersion))
			Expect(app.BuildLogs()).To(MatchRegexp("Conda Packages.*: Contributing to layer"))
		})
	})

	when("repushing the same app", func() {
		when("the app has environment.yml and a package-list.txt", func() {
			var (
				app     *dagger.App
				appRoot string
				err     error
			)

			it.Before(func() {
				appRoot = filepath.Join("testdata", "with_lock_file")
			})

			it.After(func() {
				app.Destroy()
			})

			it("reuses the packages in the launch layer using the package-list.txt", func() {
				app, err = dagger.PackBuild(appRoot, condaURI)
				Expect(err).NotTo(HaveOccurred())

				app.Env["PORT"] = "8080"
				Expect(app.Start()).To(Succeed())

				_, imageID, _, err := app.Info()
				Expect(err).NotTo(HaveOccurred())

				buildLogs := new(bytes.Buffer)
				app.BuildLogs()
				cmd := exec.Command("pack", "build", imageID, "--builder", "cfbuildpacks/cflinuxfs3-cnb-test-builder", "--buildpack", condaURI)
				cmd.Dir = appRoot
				cmd.Stdout = io.MultiWriter(os.Stdout, buildLogs)
				cmd.Stderr = io.MultiWriter(os.Stderr, buildLogs)
				Expect(cmd.Run()).To(Succeed())

				const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

				re := regexp.MustCompile(ansi)
				strippedLogs := re.ReplaceAllString(buildLogs.String(), "")

				Expect(strippedLogs).To(MatchRegexp("Conda Packages.*: Reusing cached layer"))

				body, _, err := app.HTTPGet("/")
				Expect(err).ToNot(HaveOccurred())
				Expect(body).To(ContainSubstring("Hello, world!"))
			})
		})

		when("the app has environment.yml and NO package-list.txt", func() {
			var (
				app     *dagger.App
				appRoot string
				err     error
			)

			it.Before(func() {
				appRoot = filepath.Join("testdata", "simple_app")
			})

			it.After(func() {
				app.Destroy()
			})

			it("reuses the packages in the cache layer and uses the environment.yml", func() {
				app, err = dagger.PackBuild(appRoot, condaURI)
				Expect(err).NotTo(HaveOccurred())

				app.Env["PORT"] = "8080"
				Expect(app.Start()).To(Succeed())

				_, imageID, _, err := app.Info()
				Expect(err).NotTo(HaveOccurred())

				buildLogs := new(bytes.Buffer)
				app.BuildLogs()
				cmd := exec.Command("pack", "build", imageID, "--builder", "cfbuildpacks/cflinuxfs3-cnb-test-builder", "--buildpack", condaURI)
				cmd.Dir = appRoot
				cmd.Stdout = io.MultiWriter(os.Stdout, buildLogs)
				cmd.Stderr = io.MultiWriter(os.Stderr, buildLogs)
				Expect(cmd.Run()).To(Succeed())

				const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

				re := regexp.MustCompile(ansi)
				strippedLogs := re.ReplaceAllString(buildLogs.String(), "")

				Expect(strippedLogs).NotTo(MatchRegexp("Conda Packages.*: Reusing cached layer"))
				Expect(strippedLogs).NotTo(ContainSubstring("Downloading and Extracting Packages")) // Shows that conda is caching

				body, _, err := app.HTTPGet("/")
				Expect(err).ToNot(HaveOccurred())
				Expect(body).To(ContainSubstring("Hello, world!"))
			})
		})

	})

	when("the app is vendored", func() {
		var (
			app     *dagger.App
			appRoot string
			err     error
		)

		it.Before(func() {
			appRoot = filepath.Join("testdata", "vendored")
		})

		it.After(func() {
			app.Destroy()
		})

		it("uses the vendored packages", func() {
			app, err = dagger.PackBuild(appRoot, condaURI)
			Expect(err).NotTo(HaveOccurred())
			Expect(app.BuildLogs()).To(ContainSubstring("file:///workspace/vendor"))

			app.Env["PORT"] = "8080"
			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, world!"))
		})
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
