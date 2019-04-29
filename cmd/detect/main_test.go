package main

import (
	"github.com/cloudfoundry/conda-cnb/conda"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var (
		factory         *test.DetectFactory
		envYamlContents = `
name: pydata_test
dependencies:
- numpy
`
	)

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
	})

	when("there is an environment.yml file", func() {
		it.Before(func() {
			contents := envYamlContents
			Expect(helper.WriteFile(filepath.Join(factory.Detect.Application.Root, "environment.yml"), 0666, contents)).To(Succeed())
		})

		it("still passes on an empty string for the python version in the build plan", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Output).To(HaveKeyWithValue(conda.CondaLayer,
				buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"build":  true,
						"launch": true,
					},
					Version: "",
				}))
		})
	})

	when("there is no environment.yml file", func() {
		it("should fail", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(detect.FailStatusCode))
		})
	})
}
