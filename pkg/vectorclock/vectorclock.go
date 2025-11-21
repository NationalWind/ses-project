package vectorclock

import (
	"fmt"
	"sync"
)

// VectorEntry represents (P', t) trong V_P
// P' là process ID đích, t là vector timestamp
type VectorEntry struct {
	TargetProcessID int   // P'
	Timestamp       []int // t - vector timestamp
}

// VectorClock cho thuật toán SES theo slide
// Mỗi process lưu V_P: danh sách các (P', t)
type VectorClock struct {
	entries      []VectorEntry // V_P: các cặp (process_id, timestamp)
	localTime    []int         // tP: thời gian logic hiện tại tại process này
	processID    int           // ID của process này
	numProcesses int
	mu           sync.RWMutex
}

// NewVectorClock tạo vector clock mới
// Ban đầu V_P rỗng, tP = [0, 0, ..., 0]
func NewVectorClock(processID int, numProcesses int) *VectorClock {
	return &VectorClock{
		entries:      []VectorEntry{}, // V_P ban đầu rỗng
		localTime:    make([]int, numProcesses),
		processID:    processID,
		numProcesses: numProcesses,
	}
}

// GetLocalTime trả về tP hiện tại
func (vc *VectorClock) GetLocalTime() []int {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	timeCopy := make([]int, len(vc.localTime))
	copy(timeCopy, vc.localTime)
	return timeCopy
}

// GetEntries trả về bản sao của V_P
func (vc *VectorClock) GetEntries() []VectorEntry {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	entriesCopy := make([]VectorEntry, len(vc.entries))
	for i, entry := range vc.entries {
		tsCopy := make([]int, len(entry.Timestamp))
		copy(tsCopy, entry.Timestamp)
		entriesCopy[i] = VectorEntry{
			TargetProcessID: entry.TargetProcessID,
			Timestamp:       tsCopy,
		}
	}
	return entriesCopy
}

// PrepareToSend chuẩn bị gửi message đến targetID
// Theo slide:
// 1. Gửi message M với timestamp tm = tP hiện tại, cùng V_P
// 2. Thêm (targetID, tm) vào V_P (ghi đè nếu đã tồn tại)
// 3. Tăng tP[processID]++
func (vc *VectorClock) PrepareToSend(targetID int) (tm []int, vp []VectorEntry) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// 1. tm = tP hiện tại (trước khi increment)
	tm = make([]int, len(vc.localTime))
	copy(tm, vc.localTime)

	// 2. V_P để gửi đi (không bao gồm entry cho targetID)
	vp = []VectorEntry{}
	for _, entry := range vc.entries {
		if entry.TargetProcessID != targetID {
			tsCopy := make([]int, len(entry.Timestamp))
			copy(tsCopy, entry.Timestamp)
			vp = append(vp, VectorEntry{
				TargetProcessID: entry.TargetProcessID,
				Timestamp:       tsCopy,
			})
		}
	}

	// 3. Thêm/cập nhật (targetID, tm) vào V_P của sender
	found := false
	for i := range vc.entries {
		if vc.entries[i].TargetProcessID == targetID {
			// Ghi đè
			copy(vc.entries[i].Timestamp, tm)
			found = true
			break
		}
	}
	if !found {
		// Thêm mới
		tmCopy := make([]int, len(tm))
		copy(tmCopy, tm)
		vc.entries = append(vc.entries, VectorEntry{
			TargetProcessID: targetID,
			Timestamp:       tmCopy,
		})
	}

	// 4. Increment local time: tP[processID]++
	vc.localTime[vc.processID]++

	return tm, vp
}

// CanDeliver kiểm tra điều kiện deliver message
// Theo slide:
// - Nếu V_M không chứa (receiverID, t) -> có thể deliver
// - Nếu có (receiverID, t):
//   - Nếu tm > tP[receiverID]: buffer (chưa deliver)
//   - Nếu tm <= tP[receiverID]: deliver
//
// Giải thích: t > tP nghĩa là "có sự kiện trong process khác mà P chưa cập nhật"
func (vc *VectorClock) CanDeliver(tm []int, vm []VectorEntry) (bool, string) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	// Tìm entry (receiverID, t) trong V_M
	var entryForMe *VectorEntry = nil
	for i := range vm {
		if vm[i].TargetProcessID == vc.processID {
			entryForMe = &vm[i]
			break
		}
	}

	// Nếu không có entry cho receiverID -> deliver
	if entryForMe == nil {
		return true, "no entry for receiver in V_M"
	}

	// Có entry (receiverID, t)
	// Kiểm tra: tm <= tP?
	// Theo slide: tm <= tP[receiverID] (so sánh scalar)
	// Nhưng tm là vector, nên ta cần so sánh vector

	// Cách hiểu: kiểm tra xem có dependency nào chưa thỏa mãn không
	// Nếu entryForMe.Timestamp[j] > localTime[j] cho bất kỳ j nào
	// -> có sự kiện từ process j mà ta chưa biết -> buffer

	for j := 0; j < len(entryForMe.Timestamp) && j < len(vc.localTime); j++ {
		if entryForMe.Timestamp[j] > vc.localTime[j] {
			return false, fmt.Sprintf("missing dependency from P%d: need %d, have %d",
				j, entryForMe.Timestamp[j], vc.localTime[j])
		}
	}

	return true, "all dependencies satisfied"
}

// DeliverMessage cập nhật vector clock sau khi deliver
// Theo slide: merge V_M vào V_P và cập nhật tP
func (vc *VectorClock) DeliverMessage(senderID int, tm []int, vm []VectorEntry) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// 1. Cập nhật tP với tm
	// tP = max(tP, tm) cho mỗi component
	for i := 0; i < len(vc.localTime) && i < len(tm); i++ {
		if tm[i] > vc.localTime[i] {
			vc.localTime[i] = tm[i]
		}
	}

	// 2. Increment tP[senderID] vì ta vừa nhận 1 message từ sender
	vc.localTime[senderID]++

	// 3. Merge V_M vào V_P
	// Với mỗi entry (P', t') trong V_M:
	// - Nếu V_P đã có (P', t): cập nhật t = max(t, t')
	// - Nếu chưa có: thêm mới
	for _, vmEntry := range vm {
		found := false
		for i := range vc.entries {
			if vc.entries[i].TargetProcessID == vmEntry.TargetProcessID {
				// Merge: lấy max cho mỗi component
				for j := 0; j < len(vc.entries[i].Timestamp) && j < len(vmEntry.Timestamp); j++ {
					if vmEntry.Timestamp[j] > vc.entries[i].Timestamp[j] {
						vc.entries[i].Timestamp[j] = vmEntry.Timestamp[j]
					}
				}
				found = true
				break
			}
		}
		if !found {
			// Thêm mới
			tsCopy := make([]int, len(vmEntry.Timestamp))
			copy(tsCopy, vmEntry.Timestamp)
			vc.entries = append(vc.entries, VectorEntry{
				TargetProcessID: vmEntry.TargetProcessID,
				Timestamp:       tsCopy,
			})
		}
	}
}

// String trả về string representation
func (vc *VectorClock) String() string {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	return fmt.Sprintf("tP=%v, V_P=%v", vc.localTime, vc.entries)
}

// PruneEntries loại bỏ các entry cũ không cần thiết
// (Optional optimization)
func (vc *VectorClock) PruneEntries() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// Loại bỏ entry (P', t) nếu t <= tP
	// Vì những entry này không còn cần thiết để kiểm tra causality
	newEntries := []VectorEntry{}
	for _, entry := range vc.entries {
		needKeep := false
		for j := 0; j < len(entry.Timestamp) && j < len(vc.localTime); j++ {
			if entry.Timestamp[j] > vc.localTime[j] {
				needKeep = true
				break
			}
		}
		if needKeep {
			newEntries = append(newEntries, entry)
		}
	}
	vc.entries = newEntries
}
