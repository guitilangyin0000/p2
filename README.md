### this is a lab project named Tribber in cmu 15-440, this project implements a micro blog.

## Running code

### The srunner program is storage server runner

```
    # start a single master storage server on port 9009
    ./srunner -port=9009

    # start the master on port 9009 ans run two additional slaves
    ./srunner -port=9009 -N=3
    ./srunner -master="localhost:9009"
    ./srunner -master="localhost:9009"
    (note that if you input -N=3 and you only register one slave server,the
    register won't be successful until you register another slave server)
```
in the above example you don't need to specify a port for your slave server.

### The lrunner program is libstore-runner
it runs an instance of libstore implemention. It enables you to execute lib-
store methods from the command line.

```bash
    #create one storage servers in the background
    ./srunner -port=9009 &

    #execute put("thom", "yorke")
    ./lrunner -port=9009 p thom yorke
    OK

```

### The trunner program is tribbler server runner

It creates and runs an instance of your TribServer implemention.

```bash
    #create one tribbler server on port 9010 which connect to storage server
    #localhost:9009
    ./trunner -port=9010 localhost:9009
```

### The crunner program creates and runs an instance of the TribClient
