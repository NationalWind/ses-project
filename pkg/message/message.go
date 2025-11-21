package message

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/NationalWind/ses-project/pkg/vectorclock"
)

// Message trong SES theo slide
// Bao gồm: nội dung, tm (timestamp), V_M (vector entries)
type Message struct {
	ID         string                    `json:"id"`          // Unique message ID
	SenderID   int                       `json:"sender_id"`   // ID of sender
	ReceiverID int                       `json:"receiver_id"` // ID of receiver
	Content    string                    `json:"content"`     // Message content
	Timestamp  []int                     `json:"timestamp"`   // tm: vector timestamp khi gửi
	VectorP    []vectorclock.VectorEntry `json:"vector_p"`    // V_P: các cặp (process_id, timestamp)
	PhysicalTS time.Time                 `json:"physical_ts"` // Physical timestamp (for logging)
	SeqNum     int                       `json:"seq_num"`     // Sequence number
}

type Status string

const (
	StatusSent      Status = "SENT"
	StatusReceived  Status = "RECEIVED"
	StatusBuffered  Status = "BUFFERED"
	StatusDelivered Status = "DELIVERED"
)

type MessageLog struct {
	Message   Message   `json:"message"`
	Status    Status    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Reason    string    `json:"reason,omitempty"`
}

// NewMessage tạo message mới
func NewMessage(senderID, receiverID, seqNum int, content string, tm []int, vp []vectorclock.VectorEntry) Message {
	return Message{
		ID:         fmt.Sprintf("P%d-P%d-M%d", senderID, receiverID, seqNum),
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
		Timestamp:  tm,
		VectorP:    vp,
		PhysicalTS: time.Now(),
		SeqNum:     seqNum,
	}
}

func LogMessage(msg Message, status Status, reason string) string {
	logEntry := MessageLog{
		Message:   msg,
		Status:    status,
		Timestamp: time.Now(),
		Reason:    reason,
	}
	data, _ := json.MarshalIndent(logEntry, "", "  ")
	return string(data)
}

func (m *Message) ToJSON() string {
	data, _ := json.MarshalIndent(m, "", "  ")
	return string(data)
}

func FromJSON(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

func (m *Message) Encode(conn net.Conn) error {
	return json.NewEncoder(conn).Encode(m)
}

func DecodeMessage(conn net.Conn) (Message, error) {
	var msg Message
	err := json.NewDecoder(conn).Decode(&msg)
	return msg, err
}

// Helper để format V_P cho logging
func FormatVectorP(vp []vectorclock.VectorEntry) string {
	if len(vp) == 0 {
		return "[]"
	}
	result := "["
	for i, entry := range vp {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("(P%d,%v)", entry.TargetProcessID, entry.Timestamp)
	}
	result += "]"
	return result
}
