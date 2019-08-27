.PHONY: all clean test push
all: test push

test:
	# Python 2 is required due to a problem with container using python 2
	bazel test //robots/flakefinder:*  --host_force_python=PY2

push:
	# Python 2 is required due to a problem with container using python 2
	bazel run //robots/flakefinder:push --host_force_python=PY2
	bash -x ./update-jobs-with-latest-flakefinder-image.sh
