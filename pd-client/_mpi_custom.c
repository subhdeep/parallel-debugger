#include <stdio.h>
#include <stdlib.h>
#include "mpi.h"

int MPI_Init(int *argc, char ***argv) {
  int return_code, size, rank;
  FILE* f;
  char *filename;

  printf("Preloaded.\n");
  return_code = PMPI_Init(argc, argv);
  if (return_code != MPI_SUCCESS) {
    return return_code;
  }
  MPI_Comm_rank(MPI_COMM_WORLD, &rank);
  MPI_Comm_size(MPI_COMM_WORLD, &size);
  filename = getenv("FILENAME");
  f = fopen(filename, "w");
  fprintf(f, "%d,%d\n", rank, size);
  fclose(f);
  return return_code;
}
