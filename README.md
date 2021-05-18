# Miniconda Cloud Native Buildpack

## Integration

The Miniconda CNB provides conda as a dependency. Downstream buildpacks can
require the conda dependency by generating a [Build Plan
TOML](https://github.com/buildpacks/spec/blob/master/buildpack.md#build-plan-toml)
file that looks like the following:

```toml
[[requires]]

  # The name of the Miniconda dependency is "conda". This value is considered
  # part of the public API for the buildpack and will not change without a plan
  # for deprecation.
  name = "conda"

  # The version of the conda dependency is not required. In the case it
  # is not specified, the buildpack will provide the default version, which can
  # be seen in the buildpack.toml file.
  # If you wish to request a specific version, the buildpack supports
  # specifying a semver constraint in the form of "4.*", "4.7.*", or even
  # "4.7.12".
  version = "4.7.12"

  # The Miniconda buildpack supports some non-required metadata options.
  [requires.metadata]

    # Setting the build flag to true will ensure that the conda
    # dependency is available on the $PATH for subsequent buildpacks during
    # their build phase. If you are writing a buildpack that needs to run
    # miniconda during its build process, this flag should be set to true.
    build = true

    # Setting the launch flag to true will ensure that the conda
    # dependency is available on the $PATH for the running application. If you are
    # writing an application that needs to run miniconda at runtime, this flag
    # should be set to true.
    launch = true
```

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh --version <version-number>
```

This will create a `buildpackage.cnb` file under the `build` directory which you
can use to build your app as follows:
`pack build <app-name> -p <path-to-app> -b build/buildpackage.cnb -b <other-buildpacks..>`

## Vendoring

Follow these steps to vendor python packages in your app using conda

**Prerequisites**
- Must be run on linux OS and **case-sensitive file system**
- Install conda build tools: `conda install conda-build`

**Steps**
1. `cd <my_conda_app>`
1. Create `environment.yml` file in the root of your app
1. `CONDA_PKGS_DIRS=vendor/noarch conda env create -f environment.yml -n <env_name>`
1. `conda index vendor`
1. `conda list -n <env_name> -e > package-list.txt`
1. Commit `environment.yml`, `vendor`, and `package-list.txt`
