package integration_test

import (
	"fmt"
	"github.com/cloudfoundry/dagger"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestIntegration(t *testing.T) {
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		pythonURI, condaURI string
		err                 error
	)

	it.Before(func() {
		RegisterTestingT(t)

		pythonURI, err = dagger.PackageLocalBuildpack("python-cnb", "/Users/pivotal/workspace/python-cnb")
		Expect(err).ToNot(HaveOccurred())

		condaURI, err = dagger.PackageBuildpack()
		Expect(err).ToNot(HaveOccurred())
	})

	when("we push a simple conda app", func() {
		it("builds and runs", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app"), pythonURI, condaURI)
			Expect(err).NotTo(HaveOccurred())

			app.Env["PORT"] = "8080"
			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
			}

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("numpy: 1.10.4"))
			Expect(body).To(ContainSubstring("scipy: 0.17.0"))
			Expect(body).To(ContainSubstring("sklearn: 0.17.1"))
			Expect(body).To(ContainSubstring("pandas: 0.18.0"))

			Expect(app.Destroy()).To(Succeed())
		})
	})

	when("we push a simple conda app with a buildpack.yml", func() {
		it.Focus("python version in buildpack.yml should be used", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app_with_buildpack_yml"), pythonURI, condaURI)
			Expect(err).NotTo(HaveOccurred())

			app.Env["PORT"] = "8080"
			err = app.Start()
			if err != nil {
				_, err = fmt.Fprintf(os.Stderr, "App failed to start: %v\n", err)
			}

			body, _, err := app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("numpy: 1.10.4"))
			Expect(body).To(ContainSubstring("scipy: 0.17.0"))
			Expect(body).To(ContainSubstring("sklearn: 0.17.1"))
			Expect(body).To(ContainSubstring("pandas: 0.18.0"))
			Expect(body).To(ContainSubstring("2.7.15"))

			//Expect(app.Destroy()).To(Succeed())
		})
	})
}
