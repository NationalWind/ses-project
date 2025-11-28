package process

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"github.com/NationalWind/ses-project/pkg/message"
	"github.com/NationalWind/ses-project/pkg/vectorclock"
)

type Process struct {
	ID               int
	Address          string
	Port             int
	NumProcesses     int
	VectorClock      *vectorclock.VectorClock
	MessageBuffer    []message.Message
	DeliveredMsgs    []message.Message
	SentMsgCount     map[int]int // Đếm số message đã gửi cho mỗi process
	ReceivedMsgCount map[int]int // Đếm số message đã nhận từ mỗi process
	Logger           *log.Logger
	LogFile          *os.File
	mu               sync.Mutex
	listener         net.Listener
	peers            map[int]string
}

// NewProcess tạo process mới
func NewProcess(id int, address string, port int, numProcesses int, peers map[int]string) (*Process, error) {
	logFile, err := os.OpenFile(
		fmt.Sprintf("logs/process_%d.log", id),
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0666,
	)
	if err != nil {
		return nil, err
	}
	logger := log.New(logFile, fmt.Sprintf("[P%d] ", id), log.LstdFlags)

	p := &Process{
		ID:               id,
		Address:          address,
		Port:             port,
		NumProcesses:     numProcesses,
		VectorClock:      vectorclock.NewVectorClock(id, numProcesses),
		MessageBuffer:    []message.Message{},
		DeliveredMsgs:    []message.Message{},
		SentMsgCount:     make(map[int]int),
		ReceivedMsgCount: make(map[int]int),
		Logger:           logger,
		LogFile:          logFile,
		peers:            peers,
	}

	for i := 0; i < numProcesses; i++ {
		if i != id {
			p.SentMsgCount[i] = 0
			p.ReceivedMsgCount[i] = 0
		}
	}

	logger.Printf("=== PROCESS INITIALIZED ===")
	logger.Printf("Initial State: tP=%v, V_P=[]", p.VectorClock.GetLocalTime())

	return p, nil
}

func (p *Process) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", p.Address, p.Port))
	if err != nil {
		return err
	}
	p.listener = listener

	p.Logger.Printf("Process started at %s:%d", p.Address, p.Port)
	fmt.Printf("[P%d] Started at %s:%d\n", p.ID, p.Address, p.Port)

	go p.acceptConnections()
	return nil
}

func (p *Process) Close() {
	if p.listener != nil {
		p.listener.Close()
	}
	if p.LogFile != nil {
		p.LogFile.Close()
	}
}

func (p *Process) acceptConnections() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			p.Logger.Printf("Error accepting connection: %v", err)
			continue
		}
		go p.handleConnection(conn)
	}
}

func (p *Process) handleConnection(conn net.Conn) {
	defer conn.Close()
	msg, err := message.DecodeMessage(conn)
	if err != nil {
		p.Logger.Printf("Error decoding message: %v", err)
		return
	}
	p.receiveMessage(msg)
}

func (p *Process) SendMessages(messagesPerProcess int, messagesPerMinute int) {
	var wg sync.WaitGroup
	interval := time.Minute / time.Duration(messagesPerMinute)

	p.Logger.Printf("=== STARTING TO SEND MESSAGES ===")
	p.Logger.Printf("Messages per process: %d", messagesPerProcess)
	p.Logger.Printf("Rate: %d messages/minute", messagesPerMinute)

	for targetID := 0; targetID < p.NumProcesses; targetID++ {
		if targetID == p.ID {
			continue
		}
		wg.Add(1)
		go func(target int) {
			defer wg.Done()
			p.sendToProcess(target, messagesPerProcess, interval)
		}(targetID)
	}
	wg.Wait()

	// Log final state
	p.mu.Lock()
	finalTime := p.VectorClock.GetLocalTime()
	finalVP := p.VectorClock.GetEntries()
	p.Logger.Printf("=== FINISHED SENDING ALL MESSAGES ===")
	p.Logger.Printf("Final tP: %v", finalTime)
	p.Logger.Printf("Final V_P: %v", finalVP)
	p.Logger.Printf("Buffer size: %d", len(p.MessageBuffer))
	p.Logger.Printf("Delivered: %d", len(p.DeliveredMsgs))
	p.mu.Unlock()

	fmt.Printf("[P%d] Finished sending | tP=%v | Buffer=%d | Delivered=%d\n",
		p.ID, finalTime, len(p.MessageBuffer), len(p.DeliveredMsgs))
}

func (p *Process) sendToProcess(targetID int, count int, interval time.Duration) {
	for i := 0; i < count; i++ {
		// Random delay
		time.Sleep(time.Duration(rand.Int63n(int64(interval))))

		// Chuẩn bị gửi theo thuật toán SES
		// 1. tm = tP hiện tại
		// 2. V_M = V_P (không bao gồm entry cho targetID)
		// 3. Thêm (targetID, tm) vào V_P của sender
		// 4. tP[senderID]++
		tm, vm := p.VectorClock.PrepareToSend(targetID)

		msg := message.NewMessage(p.ID, targetID, i+1, fmt.Sprintf("message %d", i+1), tm, vm)

		p.mu.Lock()
		p.SentMsgCount[targetID]++
		p.mu.Unlock()

		if err := p.sendMessage(targetID, msg); err != nil {
			p.Logger.Printf("❌ ERROR sending to P%d: %v", targetID, err)
		} else {
			p.Logger.Printf("📤 SENT to P%d: %s | tm=%v | V_M=%s",
				targetID, msg.ID, msg.Timestamp, message.FormatVectorP(msg.VectorP))
			fmt.Printf("[P%d] SENT to P%d: %s (tm=%v)\n", p.ID, targetID, msg.ID, msg.Timestamp)
		}
	}
}

func (p *Process) sendMessage(targetID int, msg message.Message) error {
	address, ok := p.peers[targetID]
	if !ok {
		return fmt.Errorf("unknown peer: %d", targetID)
	}
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return msg.Encode(conn)
}

// receiveMessage xử lý message nhận được
func (p *Process) receiveMessage(msg message.Message) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ReceivedMsgCount[msg.SenderID]++

	localTime := p.VectorClock.GetLocalTime()
	p.Logger.Printf("📥 RECEIVED from P%d: %s | tm=%v | V_M=%s | tP=%v",
		msg.SenderID, msg.ID, msg.Timestamp, message.FormatVectorP(msg.VectorP), localTime)
	fmt.Printf("[P%d] RECEIVED from P%d: %s (tm=%v, tP=%v)\n",
		p.ID, msg.SenderID, msg.ID, msg.Timestamp, localTime)

	// QUAN TRỌNG: Truyền senderID vào CanDeliver
	canDeliver, reason := p.VectorClock.CanDeliver(msg.SenderID, msg.Timestamp, msg.VectorP)

	if canDeliver {
		p.deliverMessage(msg)
		// Sau khi deliver, thử deliver các message trong buffer
		p.tryDeliverBuffered()
	} else {
		p.bufferMessage(msg, reason)
	}
}

// deliverMessage deliver message và cập nhật vector clock
func (p *Process) deliverMessage(msg message.Message) {
	beforeTime := p.VectorClock.GetLocalTime()

	p.DeliveredMsgs = append(p.DeliveredMsgs, msg)
	p.VectorClock.DeliverMessage(msg.SenderID, msg.Timestamp, msg.VectorP)

	afterTime := p.VectorClock.GetLocalTime()

	p.Logger.Printf("✅ DELIVERED: %s | tP: %v → %v", msg.ID, beforeTime, afterTime)
	fmt.Printf("[P%d] ✓ DELIVERED: %s | tP: %v → %v\n", p.ID, msg.ID, beforeTime, afterTime)
}

// bufferMessage lưu message vào buffer
func (p *Process) bufferMessage(msg message.Message, reason string) {
	p.MessageBuffer = append(p.MessageBuffer, msg)

	p.Logger.Printf("🔄 BUFFERED: %s | Reason: %s | BufferSize: %d | tP: %v",
		msg.ID, reason, len(p.MessageBuffer), p.VectorClock.GetLocalTime())
	fmt.Printf("[P%d] ⊗ BUFFERED: %s | Reason: %s | Buffer size: %d\n",
		p.ID, msg.ID, reason, len(p.MessageBuffer))
}

// tryDeliverBuffered thử deliver các message trong buffer
func (p *Process) tryDeliverBuffered() {
	delivered := true
	deliveryRound := 0

	for delivered {
		delivered = false
		deliveryRound++

		for i := 0; i < len(p.MessageBuffer); i++ {
			msg := p.MessageBuffer[i]
			// QUAN TRỌNG: Truyền senderID vào CanDeliver
			canDeliver, _ := p.VectorClock.CanDeliver(msg.SenderID, msg.Timestamp, msg.VectorP)

			if canDeliver {
				p.Logger.Printf("📦 DELIVERING FROM BUFFER (Round %d): %s", deliveryRound, msg.ID)

				p.deliverMessage(msg)
				// Remove from buffer
				p.MessageBuffer = append(p.MessageBuffer[:i], p.MessageBuffer[i+1:]...)
				delivered = true
				break // Start over
			}
		}
	}

	if deliveryRound > 1 {
		p.Logger.Printf("✨ Delivered %d rounds from buffer", deliveryRound-1)
	}
}

// GetStats trả về statistics
func (p *Process) GetStats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	return map[string]interface{}{
		"id":                p.ID,
		"local_time":        p.VectorClock.GetLocalTime(),
		"vector_p":          p.VectorClock.GetEntries(),
		"sent_messages":     p.SentMsgCount,
		"received_messages": p.ReceivedMsgCount,
		"delivered_count":   len(p.DeliveredMsgs),
		"buffered_count":    len(p.MessageBuffer),
	}
}

// WaitForCompletion chờ cho đến khi tất cả message được deliver
func (p *Process) WaitForCompletion(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		p.mu.Lock()
		bufferSize := len(p.MessageBuffer)
		expectedTotal := 0

		for _, count := range p.ReceivedMsgCount {
			expectedTotal += count
		}

		deliveredCount := len(p.DeliveredMsgs)
		p.mu.Unlock()

		if bufferSize == 0 && deliveredCount == expectedTotal {
			p.Logger.Printf("✅ COMPLETION: All %d messages delivered!", deliveredCount)
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	return fmt.Errorf("TIMEOUT: Buffer=%d, Delivered=%d, Expected=%d",
		len(p.MessageBuffer),
		len(p.DeliveredMsgs),
		p.getTotalReceived())
}

func (p *Process) getTotalReceived() int {
	total := 0
	for _, count := range p.ReceivedMsgCount {
		total += count
	}
	return total
}
