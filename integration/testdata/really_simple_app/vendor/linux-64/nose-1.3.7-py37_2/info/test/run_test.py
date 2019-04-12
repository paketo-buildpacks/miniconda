#  tests for nose-1.3.7-py37_2 (this is a generated file);
print('===== testing package: nose-1.3.7-py37_2 =====');
print('running run_test.py');
#  --- run_test.py (begin) ---
#!/usr/bin/env python

# Check that `setuptools` dependency is satisfied.

from nose.plugins.manager import DefaultPluginManager, EntryPointPluginManager

assert EntryPointPluginManager in DefaultPluginManager.__bases__
#  --- run_test.py (end) ---

print('===== nose-1.3.7-py37_2 OK =====');
print("import: 'pkg_resources'")
import pkg_resources

print("import: 'nose'")
import nose

