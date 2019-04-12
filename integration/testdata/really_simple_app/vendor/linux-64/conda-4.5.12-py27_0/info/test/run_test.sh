

set -ex



. $PREFIX/etc/profile.d/conda.sh
conda activate base
export PYTHON_MAJOR_VERSION=$(python -c "import sys; print(sys.version_info[0])")
export TEST_PLATFORM=$(python -c "import sys; print('win' if sys.platform.startswith('win') else 'unix')")
export PYTHONHASHSEED=$(python -c "import random as r; print(r.randint(0,4294967296))") && echo "PYTHONHASHSEED=$PYTHONHASHSEED"
env | sort
conda info
py.test tests -m "not integration and not installed" -vv || true
conda create -y -p ./built-conda-test-env
conda activate ./built-conda-test-env
echo $CONDA_PREFIX
[ "$CONDA_PREFIX" = "$PWD/built-conda-test-env" ] || exit 1
exit 0
