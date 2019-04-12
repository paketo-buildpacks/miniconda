

set -ex



python -V
pydoc -h
python-config --help
idle -h
python -c "import sysconfig; print sysconfig.get_config_var('CC')"
_PYTHON_SYSCONFIGDATA_NAME=_sysconfigdata_x86_64_conda_cos6_linux_gnu python -c "import sysconfig; print sysconfig.get_config_var('CC')"
exit 0
