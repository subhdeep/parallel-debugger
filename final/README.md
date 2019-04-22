# Prerequisites

To install and run our code, you need the following:
    - mpicc (we used 3.3)
    - gcc (we used 8.3)
    - go (we used 1.11.6. At least 1.11 is needed)

To build the server, `cd` into `pd-server` and run:

```sh
$ go build
```

Go will take care of the rest of the dependencies (downloading them over the internet if needed).

To build the client, `cd` into `pd-client` and run:

```sh
$ make
```

Make sure that you have `PATH` and `LD_LIBRARY_PATH` set up to point to the necessary MPI files.

# Running

In a terminal window, run

```sh
$ cd pd-server
$ ./pd-server
```

In a separate terminal window, run

```sh
$ cd pd-client
$ mpiexec -n 4 ./pd-client "<your-ip>:8080" ./your_binary
```

For ease of testing, there is a preexisting C file which is compiled into a binary, so for testing purposes, you can simply run

```sh
$ cd pd-client
$ mpiexec -n 4 ./pd-client "localhost:8080" ./test
```

