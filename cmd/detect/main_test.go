package main

import (
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
	})

	when("there is an environment.yml file", func() {
		it.Before(func() {
			Expect(helper.WriteFile(filepath.Join(factory.Detect.Application.Root, "environment.yml"), 0666, "")).To(Succeed())
		})

		it("should pass if python version is in buildplan", func() {
			factory.AddBuildPlan("python", buildplan.Dependency{
				Version: "some-python-version",
			})
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Output).To(HaveKeyWithValue("conda",
				buildplan.Dependency{
					Metadata: buildplan.Metadata{
						"build": true,
					},
				}))
		})

		it("should fail if python version is not in buildplan", func() {
			factory.AddBuildPlan("python", buildplan.Dependency{})
			code, err := runDetect(factory.Detect)
			Expect(err).NotTo(HaveOccurred())
			Expect(code).To(Equal(detect.FailStatusCode))
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
