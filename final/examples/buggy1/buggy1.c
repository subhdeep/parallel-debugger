#include <mpi.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <time.h>

/***
 * This is a buggy parallel program where
 * one of the rank does not call a
 * collective call MPI_Bcast, which leads to
 * a deadlock
 ***/

int main (int argc, char *argv[]) {

  int my_rank, w_size;

  MPI_Init(&argc, &argv);
  MPI_Comm_rank(MPI_COMM_WORLD, &my_rank);
  MPI_Comm_size(MPI_COMM_WORLD, &w_size);
  int data = 0;

  srand(time(0));
  int vic_rank = rand() % (w_size);

  if (my_rank == 0) {
    printf("vic_rank: %d\n", vic_rank);
  }

  if (my_rank != vic_rank) {
    data = 100;
    printf("rank: %d\n", my_rank);
    sleep(10)
    MPI_Bcast(&data, 1, MPI_INT, 0, MPI_COMM_WORLD);
    /* printf("data received: %d\n", data); */
  }

  MPI_Finalize();
  return 0;

}
