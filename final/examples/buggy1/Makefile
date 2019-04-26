CC      = mpicc
COPT    = -g -O0

LD      = $(CC)
LDFLAGS = $(COPT)

all: buggy1.x

%.x: %.o buggy1.o
	$(LD) $(LDFLAGS) $< -o $@

%.o: %.c 
	$(CC) $(COPT) -c $< -o $@

clean:
	rm *.o *.x
