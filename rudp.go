package rudp

import (
	"errors"
	"strconv"
)

const (
	TypeHeartBeat = iota
	TypeCorrupt
	TypeRequest
	TypeMissing
	TypeNormal
	GeneralPackage = 128
)

type Message struct {
	Next   *Message
	Tick   int
	Buffer []byte
	Id     int
}

type MessageQueue struct {
	Head *Message
	Tail *Message
}

type Package struct {
	Next   *Package
	Buffer []byte
}

type packageBuffer struct {
	Buffer []byte
	Head   *Package
	Tail   *Package
}
type RUDP struct {
	// id of last send message
	sendId          int
	currentTick     int
	sendQueue       *MessageQueue
	sendPackage     *Package
	recvQueue       *MessageQueue
	sendHistory     *MessageQueue
	lastExpiredTick int
	sendDelay       int
	expiredTick     int
	lastSendTick    int
	error           error
	sendAgain       []int
	recvIdMin       int
	recvIdMax       int
}

func (queue *MessageQueue) push(message *Message) {
	if queue.Tail == nil {
		queue.Head = message
		queue.Tail = message
	} else {
		queue.Tail.Next = message
		queue.Tail = message
	}
}

func (queue *MessageQueue) pop(id int) *Message {
	if queue.Head == nil {
		return nil

	}
	message := queue.Head
	if message.Id != id {
		return nil
	}
	queue.Head = message.Next
	message.Next = nil
	if queue.Head == nil {
		queue.Tail = nil

	}
	return message
}

// api for user
// input data receive from udp, return data should send by udp

func (r *RUDP) Send(data []byte) {
	size := len(data)
	buffer := make([]byte, size)
	copy(buffer, data)
	// log.Print(size)
	message := &Message{
		Buffer: buffer,
		Tick:   r.currentTick,
		Id:     r.sendId,
		Next:   nil,
	}
	r.sendId++
	r.sendQueue.push(message)
}

func (r *RUDP) Receive() ([]byte, error) {
	err := r.error
	if err != nil {
		r.error = nil
		return nil, err
	}
	message := r.recvQueue.pop(r.recvIdMin)
	if message == nil {
		return nil, nil
	}

	r.recvIdMin++
	if len(message.Buffer) > 0 {
		return message.Buffer, nil
	}
	return nil, errors.New("miss package" + strconv.Itoa(message.Id))
}

func (r *RUDP) Update(rawData []byte, tick int) *Package {
	r.currentTick += tick
	r.clearOutPackage()
	r.extractPackage(rawData)

	if r.currentTick >= r.lastExpiredTick+r.expiredTick {
		r.clearSendExpired(r.lastExpiredTick)
		r.lastExpiredTick = r.currentTick
	}

	if r.currentTick >= r.lastSendTick+r.sendDelay {
		r.sendPackage = r.generateOutPackage()
		r.lastSendTick = r.currentTick
		return r.sendPackage
	}
	return nil
}

func (r *RUDP) clearOutPackage() {
	r.sendPackage = nil
}

func (r *RUDP) extractPackage(rawData []byte) {
	size := len(rawData)
	for size > 0 {
		length := int(rawData[0])
		// data message
		if length > 127 {
			if size <= 1 {
				r.error = errors.New("invalid message in extract package")
				return
			}
			length = (length*256 + int(rawData[1])) & 0x7fff
			rawData = rawData[2:]
			size -= 2
		} else {
			rawData = rawData[1:]
			size -= 1
		}

		switch length {
		case TypeHeartBeat:
			//if len(r.sendAgain) == 0 {
			// request next package id (message)?
			// r.insertSendAgain(r.recvIdMin)
			//}
		case TypeCorrupt:
			r.error = errors.New("error from peer")
			return
		case TypeRequest, TypeMissing:
			if size < 2 {
				r.error = errors.New("invalid message in extract package")
				return
			}
			// parse message id
			mId := r.getMessageId(rawData)
			if TypeRequest == length {
				r.addRequest(mId)
			} else {
				r.addMissing(mId)
			}
			rawData = rawData[2:]
			size -= 2
			// normal data
		default:
			length -= TypeNormal
			if size < (length + 2) {
				r.error = errors.New("invalid message in extract package")
				return
			} else {
				mId := r.getMessageId(rawData)
				r.insertMessage(mId, rawData[2:2+length])
			}
			rawData = rawData[2+length:]
			size -= 2 + length
		}

	}
}

func (r *RUDP) clearSendExpired(tick int) {
	tmpMessage := r.sendHistory.Head
	var lastMessage *Message = nil
	for tmpMessage != nil && tmpMessage.Tick < tick {
		lastMessage = tmpMessage
		tmpMessage = tmpMessage.Next
	}
	if lastMessage != nil {
		lastMessage.Next = nil
	}
	r.sendHistory.Head = tmpMessage
	if tmpMessage == nil {
		r.sendHistory.Tail = nil
	}
}

func (r *RUDP) generateOutPackage() *Package {
	// todo : 或许内置于rudp内部？
	tmpBuffer := &packageBuffer{
		Head:   nil,
		Tail:   nil,
		Buffer: make([]byte, 0),
	}
	r.requestMissing(tmpBuffer)
	r.replyRequest(tmpBuffer)
	r.sendMessage(tmpBuffer)
	// 如果暂时没有多个数据包
	if tmpBuffer.Head == nil {
		if len(tmpBuffer.Buffer) == 0 {
			tmpBuffer.Buffer = append(tmpBuffer.Buffer, TypeHeartBeat)
		}
	}
	r.newPackage(tmpBuffer)
	return tmpBuffer.Head
}

func (r *RUDP) insertSendAgain(minId int) {
	var index, tmpId int
	for index, tmpId = range r.sendAgain {
		if tmpId == minId {
			return
		}
		if tmpId > minId {
			break
		}
	}
	if tmpId > minId {
		// 插在中间
		rear := make([]int, 0)
		rear = append(rear, r.sendAgain[index:]...)
		r.sendAgain = append(r.sendAgain[:index], minId)
		r.sendAgain = append(r.sendAgain, rear...)
	} else {
		// 插在最后
		r.sendAgain = append(r.sendAgain, minId)
	}
}

func (r *RUDP) getMessageId(rawData []byte) int {
	// 最大ID为64K即65536，循环计数0x10000
	mId := int(rawData[0])*256 + int(rawData[1])
	// recvMaxId
	// println(mId)
	// 取出 recvMaxId 大于0xffff的部分，主要是判断是不是0x10000了
	mId = mId | (r.recvIdMax &^ 0xffff)
	//println(mId)
	// 调整顺序，序列号距离过远，以 recvMaxId 为中点，将该message作为其后半部分
	if mId < r.recvIdMax-0x8000 {
		mId += 0x10000
	} else {
		// 作为其前半部分
		if mId > r.recvIdMax+0x8000 {
			mId -= 0x10000
		}
	}
	return mId
}

func (r *RUDP) addMissing(id int) {

}

func (r *RUDP) addRequest(id int) {
	r.insertSendAgain(id)
}

func (r *RUDP) insertMessage(id int, rawData []byte) {
	if id < r.recvIdMin {
		return
	}
	// 构造要插入的message
	size := len(rawData)
	buffer := make([]byte, size)
	copy(buffer, rawData)
	message := &Message{
		Buffer: buffer,
		Id:     id,
		Next:   nil,
	}
	// 判断插入位置
	if id > r.recvIdMax || r.recvQueue.Head == nil {
		r.recvQueue.push(message)
		r.recvIdMax = id
	} else {
		tmpMessage := r.recvQueue.Head
		lastMessage := &r.recvQueue.Head
		for tmpMessage != nil {
			if tmpMessage.Id == id {
				return
			}
			if tmpMessage.Id > id {
				// 插入数据包
				message.Next = tmpMessage
				*lastMessage = message
				return
			}
			lastMessage = &tmpMessage.Next
			tmpMessage = tmpMessage.Next
		}
	}
}

func (r *RUDP) requestMissing(buffer *packageBuffer) {
	id := r.recvIdMin
	message := r.recvQueue.Head
	for message != nil {
		if message.Id > id {
			for i := id; i < message.Id; i++ {
				r.packRequest(buffer, i, TypeRequest)
			}
		}
		id = message.Id + 1
		message = message.Next
	}
}

func (r *RUDP) replyRequest(buffer *packageBuffer) {
	history := r.sendHistory.Head
	for _ ,id := range r.sendAgain {
//		if id < r.recvIdMin {
//			continue
//		}
		for {
			// 要求重发的数据不存在了
			if history == nil || id < history.Id {
				r.packRequest(buffer, id, TypeMissing)
				break
			}
			if id == history.Id {
				r.packMessage(buffer, history)
				break
			}
			history = history.Next
		}
	}
	r.sendAgain = make([]int, 0)
}

func (r *RUDP) sendMessage(buffer *packageBuffer) {
	message := r.sendQueue.Head
	for message != nil {
		r.packMessage(buffer, message)
		message = message.Next
	}
	// 加入到历史已发送队列
	if r.sendQueue.Head != nil {
		if r.sendHistory.Tail == nil {
			r.sendHistory.Head = r.sendQueue.Head
			r.sendHistory.Tail = r.sendQueue.Tail

		} else {
			r.sendHistory.Tail.Next = r.sendQueue.Head
			r.sendHistory.Tail = r.sendQueue.Tail
		}
		r.sendQueue.Head = nil
		r.sendQueue.Tail = nil
	}
}

func (r *RUDP) newPackage(buffer *packageBuffer) {
	rudpPackage := &Package{
		Buffer: make([]byte, len(buffer.Buffer)),
		Next:   nil,
	}
	copy(rudpPackage.Buffer, buffer.Buffer)
	if buffer.Tail == nil {
		buffer.Tail = rudpPackage
		buffer.Head = buffer.Tail
	} else {
		buffer.Tail.Next = rudpPackage
		buffer.Tail = rudpPackage
	}
	buffer.Buffer = make([]byte, 0)
}
// ask for resend and report missing
func (r *RUDP) packRequest(buffer *packageBuffer, id int, requestTag int) {
	leftCap := GeneralPackage - len(buffer.Buffer)
	if leftCap < 3 {
		r.newPackage(buffer)
	}
	buffer.Buffer = append(buffer.Buffer, fillHeader(requestTag, id)...)
}

func (r *RUDP) packMessage(buffer *packageBuffer, message *Message) {
	leftCap := GeneralPackage - len(buffer.Buffer)
	messageSize := len(message.Buffer)
	if messageSize > GeneralPackage-4 {
		if len(buffer.Buffer) > 0 {
			r.newPackage(buffer)
		}
		//needCap:=messageSize+4
		rudpPackage := &Package{
			Buffer: make([]byte, 0),
			Next:   nil,
		}
		rudpPackage.Buffer = append(rudpPackage.Buffer, fillHeader(messageSize+TypeNormal, message.Id)...)
		rudpPackage.Buffer = append(rudpPackage.Buffer, message.Buffer...)
		if buffer.Tail == nil {
			buffer.Head = rudpPackage
			buffer.Tail = rudpPackage
		} else {
			buffer.Tail.Next = rudpPackage
			buffer.Tail = rudpPackage
		}
		return
	}
	if messageSize+4 > leftCap {
		r.newPackage(buffer)
	}

	buffer.Buffer = append(buffer.Buffer, fillHeader(messageSize+TypeNormal, message.Id)...)
	buffer.Buffer = append(buffer.Buffer, message.Buffer...)
}

func fillHeader(tag int, id int) []byte {
	retBuffer := make([]byte, 0)
	if tag < 128 {
		retBuffer = append(retBuffer, byte(tag))
	} else {
		retBuffer = append(retBuffer, byte(((tag&0x7f00)>>8)|0x80), byte(tag&0xff))
	}
	retBuffer = append(retBuffer, byte((id&0xff00)>>8), byte(id&0xff))
	return retBuffer
}

func New(sendDelay, expiredTick int) *RUDP {

	return &RUDP{
		sendDelay:   sendDelay,
		expiredTick: expiredTick,
		sendQueue:   &MessageQueue{},
		sendPackage: &Package{
			Buffer: make([]byte, 0),
		},
		recvQueue:   &MessageQueue{},
		sendHistory: &MessageQueue{},

		sendAgain: make([]int, 0),
	}
}
