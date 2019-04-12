from flask import Flask
import os
import importlib
import sys

MODULE_NAMES = []
modules = {}

for m in MODULE_NAMES:
    try:
        modules[m] = importlib.import_module(m)
    except ImportError:
        modules[m] = None

app = Flask(__name__)

def module_version(module_name):
    m = modules[module_name]
    if m is None:
        version_string = "{}: unable to import".format(module_name)
    else:
        version_string = "{}: {}".format(module_name, m.__version__)
    return version_string


@app.route('/')
def root():
    versions = "<br>\n".join([module_version(m) for m in MODULE_NAMES])
    python_version = "\npython-version%s\n" % sys.version
    r = """<br><br>
    Imports Successful!<br>

    To test each module go to /numpy, /scipy, /sklearn and /pandas
    or test all at /all.<br>
    Test suites can take up to 10 minutes to run, main output is in app logs."""
    return python_version + versions + r

if __name__ == '__main__':
    # Get port from environment variable or choose 9099 as local default
    port = int(os.getenv("PORT", 8080   ))
    # Run the app, listening on all IPs with our chosen port number
    app.run(host='0.0.0.0', port=port, debug=True)
