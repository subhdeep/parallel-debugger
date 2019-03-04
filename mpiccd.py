#!/usr/bin/env python2

from __future__ import print_function
import getopt
from subprocess import check_output, call
import sys
import os

def get_c_files(args):
    files = []
    for arg in args:
        if os.path.isfile(arg):
            output = check_output(['file', arg]).decode('utf8')
            if "C source" in output:
                files.append(arg)
    return files

def add_header_to_files(files):
    for f in files:
        lines = []
        with open(f, "r") as fl:
           lines = ['#include "debugger.h"\n'] + fl.readlines()
        with open(f, "w") as fl:
            fl.write("".join(lines))

def remove_header_from_files(files):
    for f in files:
        lines = []
        with open(f, "r") as fl:
           lines = fl.readlines()[1:]
        with open(f, "w") as fl:
            fl.write("".join(lines))

def main():
    files = get_c_files(sys.argv[1:])
    add_header_to_files(files)
    call(["mpicc"] + sys.argv[1:], shell=False, stdout=sys.stdout, stderr=sys.stderr)
    remove_header_from_files(files)

if __name__ == "__main__":
    main()
