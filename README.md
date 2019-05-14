Conda Cloud Native Buildpack
To package this buildpack for consumption:
```
$ ./scripts/package.sh
```
This builds the buildpack's Go source using GOOS=linux by default. You can supply another value as the first argument to package.sh.

# Vendoring 

Follow these steps to vendor python packages in your app using conda

**Prerequisites** 
- Must be run on linux OS and **case-insensitive file system**
- Install conda build tools: `conda install conda-build`

**Steps**
1. `cd <my_conda_app>`
1. Create `environment.yml` file in the root of your app
1. `CONDA_PKGS_DIRS=vendor/noarch conda env create -f environment.yml -n <env_name>`
1. `conda index vendor`
1. `conda list -n <env_name> -e > package-list.txt`
1. Commit `environment.yml`, `vendor`, and `package-list.txt`
