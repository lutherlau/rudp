# RUDP(Reliable UDP)

## How to use
```go
// new RUDP Object
rudp := New(1,8)

// send byte
rudp.Send([]byte{1,2,3,4}])

// receive package
recBytes, err := rudp.Receive()

// update by tick, rawData is received from udp peer, rudp Package is should send to peer
rudpPackage := rudp.Update(rawData, tick)
```

## Origin
[Blog](https://blog.codingnow.com/2016/03/reliable_udp.html)
[C Edition](https://github.com/cloudwu/rudp)
