#include "mpi.h"
#include <unistd.h>
#include <stdio.h>

int main(int argc, char* argv[]) {
  MPI_Init(&argc, &argv);
  int rank;
  int data;
  MPI_Comm_rank(MPI_COMM_WORLD, &rank);
  data = rank;
  sleep((1 + rank) * 5);
  MPI_Bcast(&data, 1, MPI_INT, 0, MPI_COMM_WORLD);
  MPI_Barrier(MPI_COMM_WORLD);
  MPI_Finalize();
  return 0;
}
