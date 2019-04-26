CC      = mpicc
COPT    = -g -O0

LD      = $(CC)
LDFLAGS = $(COPT)

all: buggy2.x

%.x: %.o buggy2.o
	$(LD) $(LDFLAGS) $< -o $@

%.o: %.c 
	$(CC) $(COPT) -c $< -o $@

clean:
	rm *.o *.x
