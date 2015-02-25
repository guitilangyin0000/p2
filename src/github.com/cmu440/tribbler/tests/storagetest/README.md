This test script is used to test storageserver
```go
portnum   = flag.Int("port", 9019, "port # to listen on")
testType  = flag.Int("type", 1, "type of test, 1: jtest, 2: btest")
numServer = flag.Int("N", 1, "(jtest only) total # of storage servers")
myID      = flag.Int("id", 1, "(jtest only) my id")
testRegex = flag.String("t", "", "test to run")
```

As code showed before, there is two type of test. The type 1 is to test
init storageserver.

First, you should open a terminal and changed directory into runner/srunner
type followed code:
```bash
./srunner -N=2 -master=9009
```
Then, you should open another terminal and changed directory tests/storage-
test, type followed code:
```bash
./storagetest -N=2 -type=1 127.0.0.1:9009
```
If you implements your code completely, you will see code
```bash
Running testInitStorageServers:
14:55:39.735197 storagetest.go:147: reply status WrongServer
PASS
Passed (1/1) tests
```

The type 2 is to test other storageserver function.
First, you should open a terminal and changed directory into runner/srunner
type followed code:
```bash
./srunner -master=9009
```
Then, you should open another terminal and changed directory tests/storage-
test, type followed code:
```bash
./storagetest -type=2 127.0.0.1:9009
```
If you implements your code completely, you will see pass all test.

