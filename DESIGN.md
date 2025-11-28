# SES Project - Design & Implementation Document

## 1. System Overview

### High-Level Architecture

```
┌────────────────────────────────────────────────────────┐
│                   SES System (15 Processes)            │
├────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐            │
│  │ Process 0│   │ Process 1│   │Process 14│ ...        │
│  │          │   │          │   │          │            │
│  │ VC: [..] │   │ VC: [..] │   │ VC: [..] │            │
│  │ Buffer:[]│   │ Buffer:[]│   │ Buffer:[]│            │
│  └────┬─────┘   └────┬─────┘   └────┬─────┘            │
│       │              │              │                  │
│       └──────────────┼──────────────┘                  │
│                      │                                 │
│          TCP Network (Localhost)                      │
│          Ports: 8000-8014, 9012-9013                 │
│                                                        │
└────────────────────────────────────────────────────────┘
```

### Core Components

1. **Vector Clock**: Maintains causal ordering information
2. **Message Buffer**: Holds messages waiting for dependencies
3. **Process Server**: TCP listener for incoming messages
4. **Sender Threads**: Goroutines that send messages to other processes
5. **Delivery Logic**: Checks dependencies and delivers messages

## 2. Vector Clock Implementation

### Data Structure

```go
type VectorClock struct {
    entries       []VectorEntry  // V_P: piggybacked entries
    localTime     []int          // tP: local vector timestamp
    processID     int
    numProcesses  int
    mu            sync.RWMutex   // Thread-safe access
}

type VectorEntry struct {
    TargetProcessID int    // P'
    Timestamp       []int  // t
}
```

**Key insight**: tP has one counter per process in the system. V_P stores "last sent to" information.

### Algorithm Operations

#### 1. Initialization
```
tP = [0, 0, 0, ..., 0]  (15 zeros for 15 processes)
V_P = []                (empty, no dependencies yet)
```

#### 2. PrepareToSend(targetID)

**Before sending message M to process P':**

```
1. tm = current tP               // Message timestamp
2. V_M = V_P except (P', t)      // Exclude entry for target
3. Add/update (P', tm) to V_P    // Record last send
4. tP[own_id]++                  // Increment own counter
```

**Example (Process 0 sending to Process 1):**
```
Before: tP=[0,0,0], V_P=[]
  Send to P1 → tm=[0,0,0], V_M=[], Add (1,[0,0,0]) to V_P
After:  tP=[1,0,0], V_P=[(1,[0,0,0])]
```

#### 3. CanDeliver(tm, V_M)

**Check if message from P_sender with tm and V_M can be delivered:**

```
IF V_M contains entry (receiverID, t):
    FOR each component j in t:
        IF t[j] > tP[j]:
            RETURN false (missing dependency from P_j)
    RETURN true
ELSE:
    RETURN true (no dependencies specified)
```

**Interpretation:**
- If sender says "I need to have processed up to t[j] from P_j", but we haven't, we must buffer
- The entry represents a "happens-before" dependency

#### 4. DeliverMessage(senderID, tm, V_M)

**After deciding to deliver:**

```
1. tP = max(tP, tm) componentwise     // Update with message timestamp
2. tP[senderID]++                     // We received from sender
3. Merge V_M into V_P:
   FOR each entry (P', t') in V_M:
       IF V_P has (P', t):
           t = max(t, t') componentwise
       ELSE:
           Add (P', t') to V_P
```

### Example Scenario

**Three processes: P0, P1, P2**

```
T0: P0 sends M1 to P1
    tP[0]: [0,0,0] → [1,0,0]
    V_P[0]: [] → [(1,[0,0,0])]

T1: P1 receives M1
    Check: V_M=[], no entry for P1, CAN DELIVER
    Deliver M1
    tP[1]: [0,0,0] → [0,1,0]  (but first merge: max([0,0,0], [1,0,0]) = [1,0,0])
    After deliver: tP[1]: [1,1,0], V_P[1]: []

T2: P0 sends M2 to P2, piggyback: V_M=[(1,[0,0,0])]
    Before: tP[0]=[1,0,0], V_P[0]=[(1,[0,0,0])]
    Send: tm=[1,0,0], V_M=[(1,[0,0,0])]
    After: tP[0]=[2,0,0], V_P[0]=[(1,[1,0,0])]

T3: P2 receives M2
    Check: V_M contains (1,[0,0,0])
    Is [0,0,0] ≤ tP[2]=[0,0,0]? YES, all components ok
    CAN DELIVER
    Deliver M2
    Merge: V_M=[(1,[0,0,0])] into V_P[2]=[]
    Result: V_P[2]=[(1,[0,0,0])]
```

## 3. Message Buffer & Delivery

### Buffering Strategy

**Messages are buffered when:**
- CanDeliver() returns false
- Dependency not yet satisfied

**Data structure:**
```go
MessageBuffer []Message  // FIFO queue
```

**Greedy delivery:**
After each message delivery, try to deliver ALL buffered messages in order.

### Buffer Processing Algorithm

```go
func tryDeliverBuffered() {
    delivered := true
    deliveryRound := 0

    for delivered {
        delivered = false
        deliveryRound++

        for i := 0; i < len(MessageBuffer); i++ {
            if canDeliver(MessageBuffer[i]) {
                deliverMessage(MessageBuffer[i])
                remove from buffer
                delivered = true
                break  // restart from beginning
            }
        }
    }
}
```

**Why restart from beginning?**
- Delivering one message might enable multiple buffered messages
- A message later in queue might become deliverable before earlier ones
- Ensures fairness and correct ordering

### Example Buffering Scenario

```
P1 receives messages in order: M_A, M_B, M_C

M_A: V_M=[(2,[2,0,0])], tP[1]=[0,0,0]
     2[0]=2 > tP[1][0]=0 → BUFFER (waiting for P2)

M_B: V_M=[], no dependencies → DELIVER immediately
     Update tP[1]=[0,1,0]

M_C: V_M=[(2,[2,0,0])], tP[1]=[0,1,0]
     Still: 2[0]=2 > tP[1][0]=0 → BUFFER

Later, P1 receives M_from_P2:
     DeliverMessage: tP[1] updated to include P2's timestamp
     Now tP[1][0] ≥ 2

TryDeliverBuffered:
     M_A: canDeliver? → YES → Deliver
     M_C: canDeliver? → YES → Deliver

Final order: B, A, C (not received order, but causally correct)
```

## 4. Network & IPC Design

### Why TCP?

| Feature | TCP | UDP | Sockets |
|---------|-----|-----|---------|
| Reliability | ✓ | ✗ | ✓ |
| In-order | ✓ | ✗ | Partial |
| Connection | ✓ | ✗ | ✓ |
| Overhead | Moderate | Low | High |

**Trade-off**: Chose TCP for reliability despite slightly higher latency.

### Connection Model

```go
// Sender side (goroutine per destination)
for each message to P_target:
    conn := DialTimeout(P_target:port, 5*time.Second)
    msg.Encode(conn)  // JSON serialize + send
    conn.Close()

// Receiver side (main server)
listener := Listen(":8000")
for {
    conn := listener.Accept()
    msg := DecodeMessage(conn)
    go receiveMessage(msg)
}
```

**Note:** New connection per message (simple but not optimized)

### Message Serialization

```go
type Message struct {
    ID         string              // "P0-P1-M5"
    SenderID   int
    ReceiverID int
    Content    string              // "message 5"
    Timestamp  []int              // sender's tP
    VectorP    []VectorEntry      // piggybacked V_M
    PhysicalTS time.Time
    SeqNum     int
}

// Encoding: JSON over TCP
json.NewEncoder(conn).Encode(msg)
```

## 5. Concurrency Model

### Process-Level Parallelism

```
Main Process (P_i)
├─ Server Goroutine (Listen & Accept)
│  └─ Handler Goroutine per connection
│     └─ receiveMessage() [locked]
│
└─ SendMessages Goroutine
   └─ For each target P_j (14 parallel)
      └─ sendToProcess(P_j)  [locked for stats]
```

### Synchronization

**Locks used:**
```go
type Process struct {
    mu sync.Mutex  // Protects:
                   // - VectorClock
                   // - MessageBuffer
                   // - DeliveredMsgs
                   // - SentMsgCount
                   // - ReceivedMsgCount
}
```

**Why single lock?**
- Simplicity (no deadlock risk)
- Acceptable contention (message processing >> lock time)
- Critical sections are short

**Lock-free operations:**
- Network I/O (happens outside locks)
- Logging (uses buffered writes)

## 6. Logging & Observability

### Log Levels & Format

```
[P0] 2025/11/27 23:30:00 === PROCESS INITIALIZED ===
[P0] 2025/11/27 23:30:00 Initial State: tP=[0 0 ...], V_P=[]
[P0] 2025/11/27 23:30:02 📤 SENT to P1: P0-P1-M1 | tm=[1 0 ...] | V_M=[...]
[P0] 2025/11/27 23:30:03 📥 RECEIVED from P1: P1-P0-M1 | tm=[...] | tP=[...]
[P0] 2025/11/27 23:30:03 ✅ DELIVERED: P1-P0-M1 | tP: [...] → [...]
[P0] 2025/11/27 23:30:03 🔄 BUFFERED: P2-P0-M1 | Reason: missing dependency from P2
[P0] 2025/11/27 23:30:05 📦 DELIVERING FROM BUFFER: P2-P0-M1
[P0] 2025/11/27 23:31:00 === FINISHED SENDING ALL MESSAGES ===
[P0] 2025/11/27 23:31:05 Final tP: [2100 ...], Final V_P: [...]
```

### What to Monitor

1. **Message Flow**: SENT → RECEIVED → DELIVERED or BUFFERED → DELIVERED
2. **Buffering Patterns**: Buffering reasons indicate causality constraints
3. **Vector Clock Evolution**: Should increase with each message processed
4. **Completion**: All messages delivered, buffer empty

## 7. Performance Characteristics

### Time Complexity

| Operation | Complexity | Notes |
|-----------|-----------|-------|
| PrepareToSend | O(V_P size) | Usually small, often < 15 |
| CanDeliver | O(vector size) | O(15) in this system |
| DeliverMessage | O(V_P size) | Merge operation |
| tryDeliverBuffered | O(buffer² × vector) | Worst case, usually fast |

### Space Complexity

| Component | Space | Notes |
|-----------|-------|-------|
| Vector Clock | O(num_processes) | 15 integers = 120 bytes |
| V_P entries | O(num_processes) | Max 15 entries × vector size |
| Message Buffer | O(num_buffered) | Unbounded, but usually small |
| Total per process | O(num_processes²) | Typically < 10 KB |

### Network Overhead

**Piggyback size per message:**
```
Base message: ~500 bytes (JSON)
Vector entries: ~15-20 × 50 bytes = 750-1000 bytes
Worst case per message: ~1.5 KB

Total for 31,500 messages: ~47.25 MB
With TCP overhead: ~55-60 MB
```

**Comparison:**
- Full Vector Clock (no filtering): ~75 MB
- SES (with selective piggyback): ~55-60 MB
- **Savings: ~25-30%**

## 8. Correctness Properties

### Safety Properties

1. **No premature delivery**
   - Message not delivered before its causal dependencies
   - Proven by CanDeliver logic

2. **Message uniqueness**
   - Each message delivered exactly once
   - No duplicates (single sequence number per sender-receiver pair)

3. **Consistency**
   - All processes eventually see same causal relationships
   - Vector clock ensures consistent interpretation

### Liveness Properties

1. **Progress**
   - Messages eventually delivered (assuming no failures)
   - Buffer can only grow if network fails

2. **Fairness**
   - Buffer processing tries all waiting messages
   - Round-robin approach prevents starvation

## 9. Implementation Highlights

### Error Handling

```go
// Network errors are logged but don't crash
if err := sendMessage(targetID, msg) {
    p.Logger.Printf("❌ ERROR sending to P%d: %v", targetID, err)
    // Continue with next message
}

// Graceful degradation
defer p.Close()  // Ensure cleanup
```

### Thread Safety

All shared state accessed through mutex:
```go
p.mu.Lock()
p.VectorClock.DeliverMessage(...)
p.DeliveredMsgs = append(...)
p.ReceivedMsgCount[msg.SenderID]++
p.mu.Unlock()
```

### Resource Management

```go
// Cleanup on exit
defer func() {
    p.listener.Close()
    p.LogFile.Close()
}()

// Bounded goroutines (max 14 senders)
for targetID := 0; targetID < p.NumProcesses; targetID++ {
    if targetID != p.ID {
        wg.Add(1)
        go func(target int) {
            defer wg.Done()
            // Limited work
        }(targetID)
    }
}
wg.Wait()  // Wait for all to complete
```

## 10. Key Design Decisions & Rationale

| Decision | Alternative | Chosen | Why |
|----------|-------------|--------|-----|
| Vector Clock filtering | Full VC | Selective piggyback | Bandwidth efficient |
| Message buffer | Discard | In-memory FIFO | Reliability |
| Synchronization | RWMutex | Single Mutex | Simplicity |
| IPC | UDP | TCP | Reliability |
| Serialization | Protobuf | JSON | Debugging, readability |
| Logging | No logging | Detailed logs | Observability |
| Process model | Single goroutine | Multi-goroutine | Concurrency |

## 11. Testing Strategy

### Unit Tests (Implicit)
- Vector clock operations correct when delivered
- Message state transitions follow rules
- Buffer processing is deterministic

### Integration Tests
- 2-3 process scenarios (verify ordering)
- Full 15-process run (verify scalability)
- Various failure scenarios (network delays, etc.)

### Observability Tests
```bash
# Verify causal ordering
grep "DELIVERED" logs/process_*.log | sort by timestamp | check order

# Verify no duplicates
grep "DELIVERED" logs/*.log | count occurrences

# Verify completeness
count(RECEIVED) ≈ count(DELIVERED)
```

## 12. Future Optimizations

1. **Connection Pooling**: Reuse TCP connections
2. **Batch Sending**: Multiple messages per TCP packet
3. **Smart Buffering**: Predict and pre-deliver messages
4. **Compression**: Compress vector clock info
5. **Pruning**: Actively remove old entries from V_P
6. **Persistent Storage**: Write logs to disk
7. **Crash Recovery**: Restore state from logs

---

**Document Version**: 1.0
**Last Updated**: 2025-11-27
**Complexity Analysis**: Suitable for distributed systems course
