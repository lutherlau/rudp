package rudp

import (
	"fmt"
)

var idx = 0

func dumpRecv(rudp *RUDP) {
	recTmp, err := rudp.Receive()
	for {
		if err != nil {
			return
		}
		if recTmp == nil {
			return
		}
		fmt.Print("RECV: ")
		if recTmp != nil {
			for _, b := range recTmp {
				fmt.Printf("0x%02x ", b)
			}
		}
		fmt.Println("")
		recTmp, err = rudp.Receive()
	}
}

func dump(p *Package) {
	fmt.Printf("%d : ", idx)
	idx++
	for p != nil {
		fmt.Print("(")
		for _, b := range p.Buffer {
			fmt.Printf("0x%02x ", b)
		}
		fmt.Print(")")
		p = p.Next
	}
	fmt.Println()
}

func main() {
	rudp := New(1, 5)
	idx = 0
	t1 := []byte{1, 2, 3, 4}
	t2 := []byte{5, 6, 7, 8}
	t3 := []byte{
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 10, 11, 12, 13,
	}

	t4 := []byte{4, 3, 2, 1}
	rudp.Send(t1)
	rudp.Send(t2)
	dump(rudp.Update(nil, 1))
	dump(rudp.Update(nil, 1))
	rudp.Send(t3)
	rudp.Send(t4)
	dump(rudp.Update(nil, 1))
	// 模拟udp接收到的数据
	r1 := []byte{02, 00, 00, 02, 00, 03}
	dump(rudp.Update(r1, 1))
	// 接受的数据
	dumpRecv(rudp)
	r2 := []byte{5, 0, 1, 1,
		5, 0, 3, 3,
	}
	dump(rudp.Update(r2, 1))
	dumpRecv(rudp)
	r3 := []byte{5, 0, 0, 0, 5, 0, 5, 5}
	dump(rudp.Update(r3, 0))
	r4 := []byte{5, 0, 2, 2}
	dump(rudp.Update(r4, 1))
	dumpRecv(rudp)
}
