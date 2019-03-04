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
    fprintf(f, "%d %s\n", i, received_hostnames[i]);
  }
  fclose(f);
}

#endif
