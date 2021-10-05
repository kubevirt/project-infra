robots/cmd/kubevirt
===================

Provides commands to manipulate the SIG testing job definitions that execute the functional tests for kubevirt/kubevirt.

Commands provided
-----------------

 * **kubevirt copy jobs**

    creates copies of the periodic and presubmit SIG jobs for latest kubevirtci providers

    Basic usage: 

        bazel run //robots/cmd/kubevirt copy jobs -- --help

 * **kubevirt require presubmits**
    
    moves presubmit job definitions for kubevirt to being required to merge
    
    Basic usage:
    
        bazel run //robots/cmd/kubevirt require presubmits -- --help

 * **kubevirt remove jobs**
    
    removes presubmit and periodic job definitions for kubevirt for unsupported kubevirtci providers
    
    Basic usage:
    
        bazel run //robots/cmd/kubevirt remove jobs -- --help

 * **kubevirt remove always_run**

    sets always_run to false on presubmit job definitions for kubevirt for unsupported kubevirtci providers

    Basic usage:

        bazel run //robots/cmd/kubevirt remove always_run -- --help

Building
--------

    make all

will build, test and check the project

See Makefile for details.
