package conda

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/runner"
	"github.com/pkg/errors"
)

const (
	CondaLayer         = "conda"
	CondaPackagesLayer = "conda_packages"
	CondaCacheLayer    = "conda_cache"
	CondaPkgsDirs      = "CONDA_PKGS_DIRS"
	EnvironmentFile    = "environment.yml"
	LockFile           = "package-list.txt"
	dependency         = "miniconda3"
	vendorDir          = "vendor"
	procfile           = "Procfile"
)

type Logger interface {
	Info(format string, args ...interface{})
}

type MetadataInterface interface {
	Identity() (name string, version string)
}

type Metadata struct {
	Name string
	Hash string
}

func (m Metadata) Identity() (string, string) {
	return m.Name, m.Hash
}

type Conda struct {
	Logger Logger
}

type Contributor struct {
	CondaPackagesMetadata MetadataInterface
	minicondaLayer        layers.DependencyLayer
	condaPackagesLayer    layers.Layer
	condaCacheLayer       layers.Layer
	context               build.Build
	runner                runner.Runner
}

func NewContributor(context build.Build, runner runner.Runner) (Contributor, bool, error) {
	willContribute := context.Plans.Has(CondaLayer)
	if !willContribute {
		return Contributor{}, false, nil
	}

	contributor := Contributor{context: context, runner: runner}

	return contributor, true, nil
}

func (c *Contributor) Contribute() error {
	if err := c.ContributeMiniconda(); err != nil {
		return errors.Wrapf(err, "unable to contribute %s layer", CondaLayer)
	}

	if err := c.ContributeCondaPackages(); err != nil {
		return errors.Wrapf(err, "unable to contribute %s layer", CondaPackagesLayer)
	}

	return c.ContributeStartCommand()
}

func (c *Contributor) ContributeMiniconda() error {
	deps, err := c.context.Buildpack.Dependencies()
	if err != nil {
		return err
	}

	dep, err := deps.Best(dependency, "*", c.context.Stack)
	if err != nil {
		return err
	}

	c.minicondaLayer = c.context.Layers.DependencyLayer(dep)

	return c.minicondaLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		if err := os.Chmod(artifact, 0777); err != nil {
			return err
		}

		return c.runner.Run(artifact, string(filepath.Separator), "-b", "-p", layer.Root)
	}, layers.Cache)
}

func (c *Contributor) ContributeCondaPackages() error {
	c.condaPackagesLayer = c.context.Layers.Layer(CondaPackagesLayer)

	if err := c.setPackagesMetadata(); err != nil {
		return err
	}

	return c.condaPackagesLayer.Contribute(c.CondaPackagesMetadata, c.packagesContribute, layers.Launch)
}

func (c *Contributor) ContributeStartCommand() error {
	procfilePath := filepath.Join(c.context.Application.Root, procfile)
	exists, err := helper.FileExists(procfilePath)
	if err != nil {
		return err
	} else if exists {
		procfileContents, err := ioutil.ReadFile(procfilePath)
		if err != nil {
			return err
		}

		procArray := strings.Split(string(procfileContents), ":")
		if len(procArray) < 2 {
			return fmt.Errorf("malformed %s", procfile)
		}

		proc := regexp.MustCompile(`^\s*web\s*:\s*`).ReplaceAllString(string(procfileContents), "")
		return c.context.Layers.WriteApplicationMetadata(layers.Metadata{Processes: []layers.Process{{"web", proc, false}}})
	}

	return nil
}

func (c *Contributor) packagesContribute(layer layers.Layer) error {
	lockFileExists, err := helper.FileExists(filepath.Join(c.context.Application.Root, LockFile))
	if err != nil {
		return err
	}

	vendorPath := filepath.Join(c.context.Application.Root, vendorDir)
	vendorDirExists, err := helper.FileExists(vendorPath)
	if err != nil {
		return err
	}

	condaBin := filepath.Join(c.minicondaLayer.Root, "bin", "conda")
	args := []string{"env", "update", "--prefix", layer.Root, "--file", EnvironmentFile}

	lockFileArgs := []string{"create", "--file", LockFile, "--prefix", layer.Root, "--yes", "--quiet"}
	vendorArgs := []string{"--channel", vendorPath, "--override-channels", "--offline"}

	if vendorDirExists {
		args = append(lockFileArgs, vendorArgs...)
		return c.runner.Run(condaBin, c.context.Application.Root, args...)
	}

	if err := c.enableCaching(); err != nil {
		return err
	}

	if lockFileExists {
		args = lockFileArgs
	}

	if err := c.runner.Run(condaBin, c.context.Application.Root, args...); err != nil {
		return err
	}

	return c.runner.Run(condaBin, c.context.Application.Root, "clean", "-pt")
}

func (c *Contributor) enableCaching() error {
	c.condaCacheLayer = c.context.Layers.Layer(CondaCacheLayer)
	c.condaCacheLayer.Touch()

	if err := os.Setenv(CondaPkgsDirs, c.condaCacheLayer.Root); err != nil {
		return err
	}

	return c.condaCacheLayer.WriteMetadata(Metadata{Name: CondaCacheLayer}, layers.Cache)
}

func (c *Contributor) setPackagesMetadata() error {
	meta := Metadata{"Conda Packages", strconv.FormatInt(time.Now().UnixNano(), 16)}

	if exists, err := helper.FileExists(filepath.Join(c.context.Application.Root, LockFile)); err != nil {
		return err
	} else if exists {
		out, err := ioutil.ReadFile(filepath.Join(c.context.Application.Root, LockFile))
		if err != nil {
			return err
		}

		hash := sha256.Sum256(out)
		meta.Hash = hex.EncodeToString(hash[:])
	}

	c.CondaPackagesMetadata = meta

	return nil
}
