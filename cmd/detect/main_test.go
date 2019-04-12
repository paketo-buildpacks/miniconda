package main

import (
	"bytes"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"path/filepath"
	"testing"

	v2Logger "github.com/buildpack/libbuildpack/logger"
	v3Logger "github.com/cloudfoundry/libcfbuildpack/logger"

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
	var factory *test.DetectFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
	})

	when("there is an environment.yml file", func() {
		when("a python version is in the buildplan and in environment.yml", func() {
			it.Before(func() {
				contents := `
name: pydata_test
dependencies:
- python=environment.yml-python-version
`
				Expect(helper.WriteFile(filepath.Join(factory.Detect.Application.Root, "environment.yml"), 0666, contents)).To(Succeed())
				factory.AddBuildPlan("python", buildplan.Dependency{
					Version: "3.6.9",
				})
			})

			it("passes detection and picks the python version in the buildplan", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(detect.PassStatusCode))
				Expect(factory.Output).To(HaveKeyWithValue("miniconda",
					buildplan.Dependency{
						Metadata: buildplan.Metadata{
							"build": true,
							"launch": true,
						},
						Version: "3.6.9",
					}))
				Expect(factory.Output).To(HaveKeyWithValue("python",
					buildplan.Dependency{
						Metadata: buildplan.Metadata{
							"build": true,
							"launch": true,
						},
						Version: "3.6.9",
					}))
			})

			it("fails to parse miniconda version from python version", func() {
				factory.AddBuildPlan("python", buildplan.Dependency{
					Version: "wut wut",
				})

				code, err := runDetect(factory.Detect)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unable to parse python major version"))
				Expect(code).To(Equal(detect.FailStatusCode))
			})

		})

		when("a python version is in the buildplan and not in environment.yml", func() {
			it.Before(func() {
				Expect(helper.WriteFile(filepath.Join(factory.Detect.Application.Root, "environment.yml"), 0666, "")).To(Succeed())
				factory.AddBuildPlan("python", buildplan.Dependency{
					Version: "2.4.6",
				})
			})

			it("passes detection and picks the python version in the buildplan", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(detect.PassStatusCode))
				Expect(factory.Output).To(HaveKeyWithValue("miniconda",
					buildplan.Dependency{
						Metadata: buildplan.Metadata{
							"build": true,
							"launch": true,
						},
						Version: "2.4.6",
					}))
				Expect(factory.Output).To(HaveKeyWithValue("python",
					buildplan.Dependency{
						Metadata: buildplan.Metadata{
							"build": true,
							"launch": true,
						},
						Version: "2.4.6",
					}))
			})
		})

		when("a python version is not in the buildplan and in environment.yml", func() {
			it.Before(func() {
				contents := `
name: pydata_test
dependencies:
- python=1.2.4
`
				Expect(helper.WriteFile(filepath.Join(factory.Detect.Application.Root, "environment.yml"), 0666, contents)).To(Succeed())
				factory.AddBuildPlan("python", buildplan.Dependency{})
			})

			it("passes detection and picks the python version in the environment.yml", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(detect.PassStatusCode))
				Expect(factory.Output).To(HaveKeyWithValue("miniconda",
					buildplan.Dependency{
						Metadata: buildplan.Metadata{
							"build": true,
							"launch": true,
						},
						Version: "1.2.4",
					}))
				Expect(factory.Output).To(HaveKeyWithValue("python",
					buildplan.Dependency{
						Metadata: buildplan.Metadata{
							"build": true,
							"launch": true,
						},
						Version: "1.2.4",
					}))
			})
		})

		when("a python version is not in the buildplan and not in environment.yml", func() {
			var (
				buf = bytes.Buffer{}
			)

			it.Before(func() {
				factory.Detect.Logger = v3Logger.Logger{Logger: v2Logger.NewLogger(&buf, &buf)}
				Expect(helper.WriteFile(filepath.Join(factory.Detect.Application.Root, "environment.yml"), 0666, "")).To(Succeed())
				factory.AddBuildPlan("python", buildplan.Dependency{})
			})

			it("fails detection and errors out", func() {
				code, err := runDetect(factory.Detect)
				Expect(err).ToNot(HaveOccurred())
				Expect(buf.String()).To(ContainSubstring("no python version specified in build plan or environment.yml"))
				Expect(code).To(Equal(detect.FailStatusCode))
			})
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
