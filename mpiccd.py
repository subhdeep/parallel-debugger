#!/usr/bin/env python2

from __future__ import print_function
import getopt
from subprocess import check_output, call
import sys
import os

debugger_filepath = "/tmp"
debugger_file = '''
#include "mpi.h"
#include <stdio.h>
#include <stdlib.h>
#ifndef _DEBUGGER_H_INCLUDED // is myheader.h already included?
#define _DEBUGGER_H_INCLUDED // define this so we know it's included

static int __rank, __w_size;

// Call this function after the MPI_Init()
void init_debugger() {
  MPI_Comm_rank(MPI_COMM_WORLD, &__rank);
  MPI_Comm_size(MPI_COMM_WORLD, &__w_size);
  char hostname[MPI_MAX_PROCESSOR_NAME];
  char received_hostnames[__w_size][MPI_MAX_PROCESSOR_NAME];
  int res, err;
  MPI_Get_processor_name(hostname, &res);
  err = MPI_Allgather(hostname, MPI_MAX_PROCESSOR_NAME, MPI_CHAR, received_hostnames, MPI_MAX_PROCESSOR_NAME, MPI_CHAR, MPI_COMM_WORLD);
  char filename[255];
  sprintf(filename, "/tmp/parallel_debugger_%d", __rank);
  FILE *f = fopen(filename, "w");
  int i;
  for (i = 0; i < __w_size; ++i) {
    fprintf(f, "%d %s\\n", i, received_hostnames[i]);
  }
  fclose(f);
}

#endif
'''

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


def make_debugger_file():
    global debugger_file
    global debugger_filepath
    with open("{}/{}".format(debugger_filepath, "debugger.h"), "w") as df:
        df.write(debugger_file)


def main():
    make_debugger_file()
    files = get_c_files(sys.argv[1:])
    add_header_to_files(files)
    sys.argv[1:] = ["-g"] + sys.argv[1:] + ["-I{}".format(debugger_filepath)]
    print(' '.join(["mpicc"] + sys.argv[1:]))
    call(["mpicc"] + sys.argv[1:], shell=False, stdout=sys.stdout, stderr=sys.stderr)
    remove_header_from_files(files)

if __name__ == "__main__":
    main()
