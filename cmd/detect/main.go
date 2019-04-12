package main

import (
	"errors"
	"fmt"
	"github.com/cloudfoundry/conda-cnb/cmd/conda"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/helper"
)

type EnvironmentYML struct {
	Name         string   `yaml:"name"`
	Dependencies []string `yaml:"dependencies"`
}

func main() {
	context, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create a default detection context: %s", err)
		os.Exit(100)
	}

	if err := context.BuildPlan.Init(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize Build Plan: %s\n", err)
		os.Exit(101)
	}

	code, err := runDetect(context)
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runDetect(context detect.Detect) (int, error) {
	environmentYMLPath := filepath.Join(context.Application.Root, "environment.yml")
	if exists, err := helper.FileExists(environmentYMLPath); err != nil {
		return detect.FailStatusCode, err
	} else if !exists {
		context.Logger.Error("no environment.yml in app root")
		return context.Fail(), nil
	}

	buildplanPyVer := context.BuildPlan[conda.PythonLayer].Version
	environmentYMLPyVer, err := readPythonVersion(environmentYMLPath)

	if err != nil {
		return detect.FailStatusCode, err
	}

	var pythonVersion string
	if buildplanPyVer != "" {
		pythonVersion = buildplanPyVer
	} else if buildplanPyVer == "" && environmentYMLPyVer != "" {
		pythonVersion = environmentYMLPyVer
	} else if buildplanPyVer == "" && environmentYMLPyVer == "" {
		context.Logger.Error("no python version specified in build plan or environment.yml")
		return context.Fail(), nil
	}

	pythonMajor, err := getMajor(pythonVersion)
	if err != nil {
		return detect.FailStatusCode, err
	}

	return context.Pass(buildplan.BuildPlan{
		conda.MinicondaLayer: buildplan.Dependency{
			Metadata: buildplan.Metadata{
				"build": true,
				"launch": true,
			},
			Version: pythonMajor,
		},
		conda.PythonLayer: buildplan.Dependency{
			Version: pythonVersion,
			Metadata: buildplan.Metadata{
				"build": true,
				"launch": true,
			},
		},
	})
}

func getMajor(s string) (string, error) {
	major := strings.Split(s, ".")[0]
	if _, err := strconv.Atoi(major); err != nil {
		return "", errors.New("unable to parse python major version")
	}
	return major, nil
}

func readPythonVersion(environmentPath string) (string, error) {
	file, err := ioutil.ReadFile(environmentPath)
	if err != nil {
		return "", err
	}

	environmentYML := EnvironmentYML{}
	err = yaml.Unmarshal(file, &environmentYML)
	if err != nil {
		return "", fmt.Errorf("oops: %v", err)
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
