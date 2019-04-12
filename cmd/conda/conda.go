package conda

import (
	"fmt"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/runner"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	MinicondaLayer = "miniconda"
	PythonLayer = "python"
	CondaModulesLayer = "condaModules"
	EnvironmentYMLLayer = "environmentYML"
)
type Logger interface {
	Info(format string, args ...interface{})
}

type Conda struct {
	Logger Logger
}


type Contributor struct {
	context build.Build
	runner runner.Runner
	launch layers.Layers
}

func NewContributor(context build.Build, runner runner.Runner) (Contributor, bool, error) {
	_, willContribute := context.BuildPlan[MinicondaLayer]
	if !willContribute {
		return Contributor{}, false, nil
	}

	contributor := Contributor{context: context, runner: runner, launch: context.Layers}

	return contributor, true, nil
}


func (c Contributor) Contribute() error {

	pythonVersion, err := c.getInstalledPythonVersion()
	if err != nil {
		return err
	}

	if err := c.ContributeMiniconda(); err != nil {
		return errors.Wrap(err, "unable to contribute miniconda layer")
	}

	if err := c.ContributeCondaModules(pythonVersion); err != nil {
		return errors.Wrap(err, "unable to contribute conda modules layer")
	}

	procFile, err := ioutil.ReadFile(filepath.Join(c.context.Application.Root, "Procfile"))
	if err != nil {
		return err
	}
	procArray := strings.Split(string(procFile), ":")

	if len(procArray) < 2 {
		return errors.New("Malformed Procfile")
	}

	proc := regexp.MustCompile(`^\s*web\s*:\s*`).ReplaceAllString(string(procFile), "")
	return c.launch.WriteApplicationMetadata(layers.Metadata{Processes: []layers.Process{{"web", proc}}})

}

func (c Contributor) flags() []layers.Flag {
	return []layers.Flag{layers.Build, layers.Launch}
}

func (c Contributor) ContributeMiniconda() error {
	deps, err := c.context.Buildpack.Dependencies()
	if err != nil {
		return err
	}

	miniconda, ok := c.context.BuildPlan[MinicondaLayer]
	if !ok {
		return errors.New("no miniconda in build plan")
	}

	pythonVersion := miniconda.Version
	pythonMajor, err := getMajor(pythonVersion)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to get python major version from python version : %s", pythonVersion))
	}

	dep, err := deps.Best("miniconda" + pythonMajor, "*", c.context.Stack)
	if err != nil {
		return err
	}

	minicondaLayer := c.context.Layers.DependencyLayer(dep)
	return minicondaLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {

		installerDir := filepath.Dir(artifact)

		dest := filepath.Join(installerDir, "conda_installer.sh")
		if err := helper.CopyFile(artifact, dest); err != nil  {
			return err
		}
		if err := os.Chmod(dest, os.ModePerm); err != nil {
			return err
		}

		if err := c.runner.Run("./conda_installer.sh", installerDir, "-b", "-p", layer.Root); err != nil {
			return err
		}

		if err := os.Setenv("PYTHONHOME", layer.Root); err != nil {
			return err
		}

		binPath := filepath.Join(minicondaLayer.Root,  "bin")
		newPath := strings.Join([]string{binPath, os.Getenv("PATH")}, string(os.PathListSeparator))
		return os.Setenv("PATH", newPath);
	}, layers.Build, layers.Launch)
}

func (c Contributor) ContributeCondaModules(pythonVersion string) error {
	condaModulesLayer := c.context.Layers.Layer(CondaModulesLayer)


	envYAMLLayer := c.context.Layers.Layer(EnvironmentYMLLayer)

	if err := os.MkdirAll(envYAMLLayer.Root, os.ModePerm); err != nil {
		return err
	}


	if err := writeNewEnvironmentYML(c.context.Application.Root, envYAMLLayer, pythonVersion); err != nil {
		return err
	}

	return condaModulesLayer.Contribute(nil, func(layer layers.Layer) error {
		environmentYmlPath := filepath.Join(envYAMLLayer.Root, "modified_environment.yml")

		if err := os.RemoveAll(layer.Root); err != nil {
			return err
		}

		args := []string{"env", "update", "-p",layer.Root, "-f", environmentYmlPath}
		if err := c.runner.Run("conda", c.context.Application.Root, args...); err != nil {
			return err
		}

		return layer.OverrideSharedEnv("PYTHONHOME", layer.Root)
	}, c.flags()...)
}

type envYMLStruct struct {
	Name         string        `yaml:"name"`
	Channels     []string      `yaml:"channels"`
	Dependencies []interface{} `yaml:"dependencies"`
}


// TODO: do we really want to have to parse this by ourselves, see TODO below
func writeNewEnvironmentYML(appRoot string, destLayer layers.Layer, pythonVersion string) error {

	fmt.Println("app Root", appRoot)
	environmentYmlPath := filepath.Join(appRoot, "environment.yml")
	envYMLContents, err := ioutil.ReadFile(environmentYmlPath)
	envYML := envYMLStruct{}
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(envYMLContents, &envYML)
	if err != nil {
		return err
	}


	replaceRe := regexp.MustCompile("python=(.*)")
	// HOPEFULLY this mutates the envYML struct :/
	for index, dep := range envYML.Dependencies {
		// TODO can the dep look different ex " python ", " Python", ...
		if val, ok := dep.(string); ok {
			replaceDep := replaceRe.ReplaceAll([]byte(val), []byte(pythonVersion))
			fmt.Printf("dep string |%s| replace Dep |%s|\n", dep, string(replaceDep))
			if string(replaceDep) != val {

				envYML.Dependencies[index] = fmt.Sprintf("python=%s", replaceDep)
			}
		}
	}

	newEnvYMLContents, err := yaml.Marshal(envYML)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(destLayer.Root, "modified_environment.yml"), newEnvYMLContents, os.ModePerm)
}

func (c *Contributor) getInstalledPythonVersion() (string, error) {
	regularExpression := `v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?`
	r := regexp.MustCompile(regularExpression)

	pythonOut, err := c.runner.RunWithOutput("python", c.context.Application.Root, "--version")
	if err != nil {
		return "", err
	}
	pythonVersion := r.Find(pythonOut)

	return string(pythonVersion), nil
}


func getMajor(s string) (string, error) {
	major := strings.Split(s, ".")[0]
	if _, err := strconv.Atoi(major); err != nil {
		return "", errors.New("unable to parse python major version")
	}
	return major, nil
}