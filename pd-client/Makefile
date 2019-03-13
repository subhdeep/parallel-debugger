CC=mpicc
CFLAGS=-g -shared -fPIC

.PHONY: all

all: mpic.so gobuild

mpic.so: _mpi_custom.c
	$(CC) $(CFLAGS) _mpi_custom.c -o mpic.so

gobuild:
	go build

clean:
	rm -f mpic.so
