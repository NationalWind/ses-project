package vectorclock

import (
	"fmt"
	"sync"
)

// VectorEntry represents (P', t) trong V_P
type VectorEntry struct {
	TargetProcessID int   // P'
	Timestamp       []int // t - vector timestamp
}

// VectorClock cho thuật toán SES
type VectorClock struct {
	entries      []VectorEntry // V_P: các cặp (process_id, timestamp)
	localTime    []int         // tP: thời gian logic hiện tại
	processID    int           // ID của process này
	numProcesses int
	mu           sync.RWMutex
}

func NewVectorClock(processID int, numProcesses int) *VectorClock {
	return &VectorClock{
		entries:      []VectorEntry{},
		localTime:    make([]int, numProcesses),
		processID:    processID,
		numProcesses: numProcesses,
	}
}

func (vc *VectorClock) GetLocalTime() []int {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	timeCopy := make([]int, len(vc.localTime))
	copy(timeCopy, vc.localTime)
	return timeCopy
}

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
// Theo SES algorithm:
// 1. Gửi message với tm = tP hiện tại và V_P (không bao gồm entry cho target)
// 2. Thêm/update (targetID, tm) vào V_P của sender
// 3. Increment tP[senderID]++
func (vc *VectorClock) PrepareToSend(targetID int) (tm []int, vp []VectorEntry) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// 1. tm = tP hiện tại (TRƯỚC khi increment)
	tm = make([]int, len(vc.localTime))
	copy(tm, vc.localTime)

	// 2. Chuẩn bị V_P để gửi (KHÔNG bao gồm entry cho targetID)
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

	// 3. Cập nhật V_P: thêm/update (targetID, tm)
	// Entry này KHÔNG được gửi trong message, nhưng được lưu local
	found := false
	for i := range vc.entries {
		if vc.entries[i].TargetProcessID == targetID {
			copy(vc.entries[i].Timestamp, tm)
			found = true
			break
		}
	}
	if !found {
		tmCopy := make([]int, len(tm))
		copy(tmCopy, tm)
		vc.entries = append(vc.entries, VectorEntry{
			TargetProcessID: targetID,
			Timestamp:       tmCopy,
		})
	}

	// 4. Increment tP[senderID]++ (sau khi set tm)
	vc.localTime[vc.processID]++

	return tm, vp
}

// CanDeliver kiểm tra điều kiện deliver theo SES algorithm
// Theo slide SES:
// 1. Tìm entry (receiverID, t) trong V_M
// 2. Nếu KHÔNG có entry → có thể deliver
// 3. Nếu có entry (receiverID, t):
//   - Nếu t >= tP: buffer (có message dependency chưa đến)
//   - Nếu t < tP: deliver (mọi dependency đã satisfied)
//
// Giải thích: entry (receiverID, t) trong V_M có nghĩa là
// "sender đã gửi message khác đến receiverID với timestamp t"
// Nếu t >= tP[receiverID], có nghĩa là receiver chưa nhận message đó
func (vc *VectorClock) CanDeliver(senderID int, tm []int, vm []VectorEntry) (bool, string) {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	// 1. Tìm entry (receiverID, t) trong V_M
	var entryForMe *VectorEntry = nil
	for i := range vm {
		if vm[i].TargetProcessID == vc.processID {
			entryForMe = &vm[i]
			break
		}
	}

	// 2. Nếu KHÔNG có entry cho receiver → có thể deliver
	if entryForMe == nil {
		return true, "no dependency"
	}

	// 3. Có entry (receiverID, t) → kiểm tra t với tP
	// Điều kiện deliver: t < tP (component-wise)
	// Nghĩa là: ∀j: t[j] <= tP[j], và tồn tại ít nhất 1 j: t[j] < tP[j]
	// HOẶC đơn giản hơn: NOT(t >= tP)

	// Kiểm tra xem có component nào của t > tP không
	for j := 0; j < len(entryForMe.Timestamp) && j < len(vc.localTime); j++ {
		if entryForMe.Timestamp[j] > vc.localTime[j] {
			// t >= tP (ít nhất 1 component) → BUFFER
			return false, fmt.Sprintf("dependency not satisfied: entry has t[%d]=%d > tP[%d]=%d",
				j, entryForMe.Timestamp[j], j, vc.localTime[j])
		}
	}

	// Tất cả components: t[j] <= tP[j] → DELIVER
	return true, "all dependencies satisfied"
}

// DeliverMessage cập nhật vector clock sau khi deliver message
// Theo SES algorithm:
// 1. Cập nhật tP theo quy tắc vector clock:
//   - tP = max(tP, tm) component-wise
//   - tP[senderID]++ (vì đã nhận 1 message từ sender)
//
// 2. Merge V_M vào V_P
func (vc *VectorClock) DeliverMessage(senderID int, tm []int, vm []VectorEntry) {
	vc.mu.Lock()
	defer vc.mu.Unlock()

	// 1. Cập nhật tP = max(tP, tm) component-wise
	for i := 0; i < len(vc.localTime) && i < len(tm); i++ {
		if tm[i] > vc.localTime[i] {
			vc.localTime[i] = tm[i]
		}
	}

	// 2. Increment tP[senderID]++ (quy tắc vector clock khi nhận message)
	vc.localTime[senderID]++

	// 3. Merge V_M vào V_P
	// Với mỗi entry (P', t') trong V_M:
	// - Nếu V_P có (P', t): cập nhật t = max(t, t') component-wise
	// - Nếu không: thêm (P', t') vào V_P
	for _, vmEntry := range vm {
		found := false
		for i := range vc.entries {
			if vc.entries[i].TargetProcessID == vmEntry.TargetProcessID {
				// Merge: component-wise max
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
			// Thêm entry mới
			tsCopy := make([]int, len(vmEntry.Timestamp))
			copy(tsCopy, vmEntry.Timestamp)
			vc.entries = append(vc.entries, VectorEntry{
				TargetProcessID: vmEntry.TargetProcessID,
				Timestamp:       tsCopy,
			})
		}
	}
}

func (vc *VectorClock) String() string {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	return fmt.Sprintf("tP=%v, V_P=%v", vc.localTime, vc.entries)
}

func (vc *VectorClock) PruneEntries() {
	vc.mu.Lock()
	defer vc.mu.Unlock()

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
