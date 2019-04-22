#include <stdio.h>
#include <stdlib.h>
#include "mpi.h"
#define _GENERATE_INTERNAL_METHOD(mname) \
  int internal_##mname() {               \
    __x = 0;                             \
    return 0;                            \
  }

#define _GENERATE_EXT_METHOD(mname, typeargs, args) \
  int mname typeargs {                              \
  int rank = -1;                                    \
  MPI_Comm_rank(comm, &rank);                       \
  __x = rank;                                       \
  internal_##mname();                               \
  volatile int result = P##mname args;              \
  return result;                                    \
  }

static volatile int __x;

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


_GENERATE_INTERNAL_METHOD(MPI_Barrier);
_GENERATE_EXT_METHOD(MPI_Barrier,(MPI_Comm comm), (comm));

_GENERATE_INTERNAL_METHOD(MPI_Bcast);
_GENERATE_EXT_METHOD(MPI_Bcast,
                     (void* data, int count, MPI_Datatype datatype, int root, MPI_Comm comm),
                     (data, count, datatype, root, comm));

/* int MPI_Barrier(MPI_Comm comm) { */
/*   int rank = -1; */
/*   MPI_Comm_rank(comm, &rank); */
/*   __x = rank; // prevent optimizing away */
/*   internal_MPI_Barrier(); */
/*   volatile int result = PMPI_Barrier(comm); */
/*   return result; */
/* } */
