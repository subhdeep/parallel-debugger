#include "mpi.h"
#include <stdio.h>

int main(int argc, char* argv[]) {
  MPI_Init(&argc, &argv);
  printf("Running cool stuff\n");
  MPI_Finalize();
  return 0;
}
