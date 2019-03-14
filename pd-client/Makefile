CC=mpicc
CFLAGS=-g -shared -fPIC

.PHONY: all

all: mpic.so gobuild test

mpic.so: _mpi_custom.c
	$(CC) $(CFLAGS) _mpi_custom.c -o mpic.so

gobuild:
	go build

test: _test.c
	$(CC) -g _test.c -o test

clean:
	rm -f mpic.so
