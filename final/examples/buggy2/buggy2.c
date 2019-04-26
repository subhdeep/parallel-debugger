#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>

/***
 * This is a buggy parallel program where
 * one of the rank sends a data size more than
 * expected using MPI_Send/MPI_Recv
 ***/

int main (int argc, char *argv[]) {

  int my_rank, w_size;
  MPI_Status status;

  MPI_Init(&argc, &argv);
  MPI_Comm_rank(MPI_COMM_WORLD, &my_rank);
  MPI_Comm_size(MPI_COMM_WORLD, &w_size);

  int *send_data, *recv_data;

  send_data = (int *)malloc(20*sizeof(int));
  recv_data = (int *)malloc(20*sizeof(int));

  int count = 20;
  if (my_rank != 0) {
    if (my_rank == 1) count = 30;
    MPI_Send(send_data, count, MPI_INT, 0, 0, MPI_COMM_WORLD);
  }

  if (my_rank == 0) {
    for (int i = 1; i < w_size; i++) {
      MPI_Recv(recv_data, count, MPI_INT, i, 0, MPI_COMM_WORLD, &status);
      printf("Received data from %d\n", i);
    }
  }

  MPI_Finalize();
  return 0;
}
