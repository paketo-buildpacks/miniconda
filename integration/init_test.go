package integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var settings struct {
	Buildpacks struct {
		BuildPlan struct {
			Online string
		}
		Miniconda struct {
			Online  string
			Offline string
		}
	}
	Buildpack struct {
		ID   string
		Name string
	}
	Config struct {
		BuildPlan string `json:"build-plan"`
	}
}

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect

	file, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())

	Expect(json.NewDecoder(file).Decode(&settings.Config)).To(Succeed())
	Expect(file.Close()).To(Succeed())

	file, err = os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.DecodeReader(file, &settings)
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())

	root, err := filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()

	settings.Buildpacks.Miniconda.Online, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.Miniconda.Offline, err = buildpackStore.Get.
		WithVersion("1.2.3").
		WithOfflineDependencies().
		Execute(root)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.BuildPlan.Online, err = buildpackStore.Get.
		Execute(settings.Config.BuildPlan)
	Expect(err).ToNot(HaveOccurred())

	SetDefaultEventuallyTimeout(10 * time.Second)

	fmt.Println("Setup completed. Waiting...")
	time.Sleep(60 * time.Minute)
	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Default", testDefault)
	suite("LayerReuse", testReusingLayerRebuild)
	suite("Logging", testLogging)
	suite("Offline", testOffline)
	suite.Run(t)
}
