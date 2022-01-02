# TODO - pyparsing is just an example library. I used it because I don't have
# it installed on my main Python installation. The test that uses this project
# should be executed on a machine without pyparsing installed. Otherwise, there
# is a possibility that the test will return a false negative. We're testing
# here, whether regolith runs this test on a virtualenv. That virtualenv should
# have pyparsing installed. The creation of the virtualenv is also a part of
# the test. Maybe we should replace pyparsing with something else. The lighter
# the module, the better but it can't be installed on the main Python
# installation.
import pyparsing

print("Successfully imported 'pyparsing' module!")