# SES Project - Setup & Execution Guide

## Quick Start (5 minutes)

### 1. Prerequisites Check
Verify you have Go installed:
```bash
go version
# Should output: go version go1.21+ ...
```

If not installed, download from [golang.org](https://golang.org/dl)

### 2. Build the Project
```bash
cd /path/to/ses-project
go build -o ses.exe cmd/main.go
```

Expected output: No errors, `ses.exe` file created

### 3. Run the Demo
```bash
bash send_all.sh
```

This will:
- Build the project (if not done)
- Start all 15 processes simultaneously
- Each process sends messages and exit automatically
- Total runtime: ~45-60 seconds
- Logs saved to `logs/` directory

## Detailed Installation

### Step 1: Verify Go Installation

**macOS:**
```bash
brew install go@1.21
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get update
sudo apt-get install golang-go
```

**Windows:**
Download and run installer from [golang.org/dl](https://golang.org/dl)

### Step 2: Verify Project Structure
```bash
ls -la
# Expected structure:
# cmd/main.go
# pkg/message/message.go
# pkg/process/process.go
# pkg/vectorclock/vectorclock.go
# config/config.json
# send_all.sh
# README.md
```

### Step 3: Build Configuration (Optional)

The default configuration in `config/config.json` is already optimized for 15 processes.

To customize:
```json
{
    "num_processes": 15,              // Change to 3, 5, 10, etc. for testing
    "messages_per_process": 150,      // Messages per destination (150 recommended)
    "messages_per_minute": 100,       // Send rate (100-200 recommended)
    "processes": [
        // 15 process definitions with ports 8000-8014, 9012-9013
    ]
}
```

**Key parameters:**
- `num_processes`: Must match process list length
- `messages_per_process`: Total messages = num_processes × (num_processes-1) × messages_per_process
- `messages_per_minute`: Higher = faster execution, lower = more realistic delays

### Step 4: Compilation

```bash
# Standard build
go build -o ses.exe cmd/main.go

# With optimizations (faster)
go build -ldflags="-s -w" -o ses.exe cmd/main.go

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o ses cmd/main.go
GOOS=windows GOARCH=amd64 go build -o ses.exe cmd/main.go
```

Verify binary:
```bash
./ses.exe 0 &
# Should show: [P0] Process started successfully!
pkill -f "./ses.exe"
```

## Execution Modes

### Mode 1: Automated (Recommended for Demo)

**Command:**
```bash
bash send_all.sh
```

**What it does:**
- Launches all 15 processes in background
- Each process auto-sends 150 messages to 14 others
- Waits for all processes to complete
- Returns when done

**Monitoring:**
```bash
# In another terminal, watch the progress
tail -f logs/process_0.log | grep "SENT\|DELIVERED\|BUFFERED"

# Or check statistics
watch -n 1 'grep -c "DELIVERED" logs/process_0.log'
```

**Total duration:** 45-75 seconds

### Mode 2: Interactive (For Testing)

**Terminal 1 - Start a process:**
```bash
./ses.exe 0
# Output: [P0] Process started successfully!
# Waiting for commands...
```

**Commands available:**
- `s` - Start sending messages
- `i` - Show statistics
- `b` - Show buffered count
- `v` - Show vector clock
- `q` - Quit

**Example session:**
```
[P0] Process started successfully!
> s
[P0] Auto sending messages...
[P0] SENT to P1: P0-P1-M1 (tm=[1 0 0 ...])
[P0] SENT to P2: P0-P2-M1 (tm=[2 0 0 ...])
...
> i

=== Process Statistics ===
Process ID: 0
Vector Clock: [2100 ...
Delivered Messages: 1400
Buffered Messages: 0
...
> q
Shutting down...
```

### Mode 3: Multi-Terminal Interactive

**Setup (8+ terminals):**
```bash
# Terminal 1
./ses.exe 0

# Terminal 2
./ses.exe 1

# Terminal 3
./ses.exe 2

# ... etc for each process
```

Then in any terminal, type `s` to start that process sending messages.

**Advantages:**
- See individual process behavior
- Monitor specific processes
- Easy to pause/stop individual processes

**Disadvantages:**
- Requires many terminals
- Manual coordination needed
- Harder to test full system

## Post-Execution Analysis

### 1. Check Execution Summary
```bash
# Count total messages per process
for i in {0..14}; do
  sent=$(grep -c "📤 SENT" logs/process_$i.log)
  recv=$(grep -c "📥 RECEIVED" logs/process_$i.log)
  delivered=$(grep -c "✅ DELIVERED" logs/process_$i.log)
  buffered=$(grep -c "🔄 BUFFERED" logs/process_$i.log)
  printf "P%-2d: SENT=%4d RECV=%4d DELIVERED=%4d BUFFERED=%4d\n" \
    $i $sent $recv $delivered $buffered
done
```

### 2. Verify Correctness
```bash
# Check final vector clock state
tail -1 logs/process_0.log | grep "Final"

# Verify no messages stuck in buffer
grep "BUFFERED" logs/*.log | wc -l
# Should be 0 (or only during execution, not at end)

# Check for errors
grep "Error\|ERROR" logs/*.log
# Should have minimal/no errors
```

### 3. Analyze a Specific Scenario
```bash
# Find messages that were buffered
grep "BUFFERED" logs/process_0.log | head -3

# See when they were delivered
grep "DELIVERING FROM BUFFER" logs/process_0.log

# Trace dependencies
grep "P1-P0" logs/process_0.log | head -5
```

### 4. Generate Statistics
```bash
# Total statistics across all processes
echo "=== System Statistics ==="
echo "Total SENT: $(grep -c 'SENT' logs/*.log)"
echo "Total DELIVERED: $(grep -c 'DELIVERED' logs/*.log)"
echo "Total BUFFERED: $(grep -c 'BUFFERED' logs/*.log)"

# Per-process statistics
echo ""
echo "=== Per-Process Statistics ==="
for i in {0..14}; do
  delivered=$(grep -c 'DELIVERED' logs/process_$i.log)
  echo "Process $i delivered: $delivered"
done | column -t
```

## Troubleshooting

### Problem: "command not found: go"
**Solution:**
- Install Go: `brew install go` (macOS) or apt-get
- Add to PATH: `export PATH=$PATH:/usr/local/go/bin`

### Problem: "bind: address already in use"
**Solution 1:** Wait for ports to free up
```bash
# Check which process is using the port
lsof -i :8000

# Kill the process
kill -9 <PID>
```

**Solution 2:** Change port numbers
Edit `config/config.json` and change port numbers to 9000-9014

### Problem: "connection refused" errors
**Likely causes:**
1. Receiver process hasn't started yet (normal, will retry)
2. Port is blocked by firewall
3. Process crashed

**Solution:**
```bash
# Check if processes are running
ps aux | grep ses.exe | grep -v grep

# Check open ports
lsof -i :8000-8014

# Review error logs
grep -i "error\|refused" logs/*.log
```

### Problem: Very few messages delivered (< 1000)
**Likely causes:**
1. Processes exiting too quickly
2. Network latency issues
3. Send rate too slow

**Solutions:**
1. Increase `messages_per_minute` in config (try 200-500)
2. Increase `messages_per_process` in config (try 300-500)
3. Modify `send_all.sh` to wait longer:
   ```bash
   sleep 120  # Wait 2 minutes instead of 60 seconds
   ```

### Problem: Process hangs or doesn't start
**Debug steps:**
```bash
# Run single process with detailed output
./ses.exe 0

# Check logs for errors
cat logs/process_0.log | tail -20

# Try with verbose output
strace ./ses.exe 0  # (Linux only)
```

## Performance Tuning

### For Maximum Message Volume
```json
{
    "messages_per_minute": 500,  // Very fast rate
    "messages_per_process": 500  // More messages
}
```

### For Realistic Simulation
```json
{
    "messages_per_minute": 60,   // ~1 msg/sec
    "messages_per_process": 150  // Standard
}
```

### For Quick Demo (3 processes)
```json
{
    "num_processes": 3,
    "messages_per_minute": 300,
    "messages_per_process": 50
}
```

## Next Steps

1. **Run the demo**: `bash send_all.sh`
2. **Analyze logs**: Check `logs/` directory
3. **Study the algorithm**: Read README.md section on SES
4. **Modify code**: Experiment with changes
5. **Create video**: Record a run with narration

## Further Documentation

- **README.md**: Algorithm explanation, system design
- **pkg/vectorclock/vectorclock.go**: Algorithm implementation with comments
- **pkg/process/process.go**: Message handling logic
- **config/config.json**: Configuration file reference

---

**For support or issues**: Check the troubleshooting section above or review the detailed README.md
