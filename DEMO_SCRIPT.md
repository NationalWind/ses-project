# SES Project - Demo Video Script

## Overview
This script provides a guide for recording a ~5-10 minute video demonstrating the SES (Sequential Execution System) algorithm implementation.

**Target Audience**: Instructors and students in distributed systems courses

---

## Pre-Recording Checklist

- [ ] Build project: `go build -o ses.exe cmd/main.go`
- [ ] Verify no processes running: `pkill -9 ses.exe`
- [ ] Clear logs: `rm -rf logs/`
- [ ] Test run completed successfully
- [ ] Terminal window at least 100x40 characters
- [ ] Font size readable (16pt+)
- [ ] Microphone working and tested
- [ ] Recording software ready (OBS, ScreenFlow, etc.)
- [ ] Screen recording at 1080p or higher

---

## Part 1: Introduction (1 minute)

**Narration:**
> "Hello, this is a demonstration of the SES - Sequential Execution System - algorithm for distributed message ordering. This project implements a 15-process distributed system that exchanges messages while maintaining causal ordering using vector clocks.
>
> The problem we're solving: In distributed systems, messages from multiple senders can arrive out of order at a receiver. Simply processing them as they arrive can violate causality - we might process a message before processing other messages it depends on.
>
> The SES algorithm provides an elegant solution using piggybacked vector clock information, ensuring messages are delivered in causally consistent order while minimizing network overhead."

**On Screen:**
- Show the README.md file
- Highlight "Problem Being Solved" section
- Show system architecture diagram

---

## Part 2: System Setup (1-2 minutes)

**Narration:**
> "Let's look at the system configuration. We have 15 processes running on localhost, using ports 8000 through 8014. Each process will send 150 messages to each of the other 14 processes, for a total of over 31,000 messages.
>
> The configuration file specifies process IDs, network addresses, and ports. We're using TCP for reliable message delivery."

**On Screen:**
```bash
# Show the configuration
cat config/config.json
```

**Points to highlight:**
- 15 processes with unique IDs
- Localhost with different ports
- 150 messages per destination
- 100 messages/minute send rate

---

## Part 3: Building & Starting (1 minute)

**Narration:**
> "First, we'll build the project. The system is written in Go, which gives us efficient concurrency with goroutines."

**On Screen:**
```bash
go build -o ses.exe cmd/main.go
```

Wait for build to complete.

**Narration:**
> "Now we'll start all 15 processes simultaneously. Each process will automatically send messages to all other processes."

**On Screen:**
```bash
bash send_all.sh
```

Show initial output:
```
Starting all 15 processes with auto-send mode...
Starting process 0...
Starting process 1...
...
```

---

## Part 4: Message Exchange (2-3 minutes)

**Narration:**
> "Watch as messages are being sent and received. Notice the console output shows several types of events:
>
> First, we see 'SENT' messages - these are messages being transmitted from one process to another with their vector timestamp.
>
> Then we see 'RECEIVED' messages - these arrive at the destination with the sender's current vector clock state.
>
> Most importantly, we see 'DELIVERED' messages - these are messages that have been delivered to the application. A message is only delivered when all its causal dependencies have been satisfied.
>
> Importantly, we also see some 'BUFFERED' messages. These are messages that arrived before their dependencies were satisfied, so they're being held in a buffer."

**On Screen:**
Open another terminal and show live log:
```bash
tail -f logs/process_0.log | grep -E "SENT|RECEIVED|DELIVERED|BUFFERED"
```

**Point out:**
- SENT: Message headers with timestamps like "P0-P1-M5"
- RECEIVED: Messages arriving with vector information
- DELIVERED: Messages processed in order
- BUFFERED: Messages waiting, with reasons like "missing dependency from P2"

**Narration:**
> "Look at this example: Process 0 has buffered a message from Process 2 because we haven't yet seen all the messages Process 2 that Process 2 was dependent on. Once those dependencies arrive, this message will automatically be delivered from the buffer."

**On Screen:**
```bash
grep "BUFFERED" logs/process_0.log | head -3
grep "DELIVERING FROM BUFFER" logs/process_0.log | head -3
```

---

## Part 5: Vector Clock Evolution (2 minutes)

**Narration:**
> "Let's examine the vector clocks. Each process maintains a vector clock - one counter for each process in the system. Initially, all are zero.
>
> As the system runs, these clocks track causality. For example, when Process 0 sends a message, its clock increments. When other processes receive messages from Process 0, they update their knowledge of Process 0's progress."

**On Screen:**
```bash
# Show a message with full vector clock info
grep "SENT" logs/process_0.log | head -1
```

Highlight:
- Message ID: P0-P1-M1
- Timestamp (tm): [1 0 0 ...] - Process 0's vector clock at send time
- Vector_P entries (V_M): [(P1, [1 0 0...]), ...] - piggybacked information

**Narration:**
> "See the 'V_M' field? That stands for 'Vector Message' - it contains 'piggybacked' information about what other messages we've sent. This is the key innovation of the SES algorithm.
>
> Instead of sending the entire 15-component vector clock with every message, we only send essential dependency information. This reduces network overhead significantly."

**On Screen:**
Show vector clock growth:
```bash
echo "Final state of Process 0:"
tail -5 logs/process_0.log | head -3
```

---

## Part 6: Buffering Demonstration (1-2 minutes)

**Narration:**
> "Now let's look at a specific buffering scenario to understand how the algorithm ensures causality.
>
> Imagine Process 1 receives a message from Process 2, and Process 2 indicates a dependency on Process 3. If Process 1 hasn't yet seen the required messages from Process 3, this message gets buffered."

**On Screen:**
```bash
# Find a buffering example
echo "=== Buffered Messages in Process 0 ==="
grep "BUFFERED" logs/process_0.log | head -5

echo ""
echo "=== When they were delivered from buffer ==="
grep "DELIVERING FROM BUFFER" logs/process_0.log | head -5
```

**Narration:**
> "Notice how messages are being delivered from the buffer later. The system is greedy - whenever we receive a new message, we immediately try to deliver any buffered messages that now have their dependencies satisfied.
>
> This ensures:
> 1. No message is delivered before its dependencies
> 2. No unnecessary buffering - messages are delivered as soon as possible
> 3. The application sees a causally consistent stream of events"

---

## Part 7: Final Statistics (1 minute)

**Narration:**
> "Let's check the final statistics to verify correctness. All processes should have completed successfully."

**On Screen:**
```bash
echo "=== Verification of Message Delivery ==="
echo ""
for i in {0..14}; do
  sent=$(grep "SENT" logs/process_$i.log | wc -l)
  delivered=$(grep "DELIVERED" logs/process_$i.log | wc -l)
  buffered=$(grep "BUFFERED" logs/process_$i.log | wc -l)
  printf "P%-2d: SENT=%4d | DELIVERED=%4d | BUFFERED NOW=%4d\n" \
    $i $sent $delivered $buffered
done

echo ""
echo "=== System Totals ==="
echo "Total messages sent: $(grep 'SENT' logs/*.log | wc -l)"
echo "Total messages delivered: $(grep 'DELIVERED' logs/*.log | wc -l)"
echo "Total messages buffered during run: $(grep 'BUFFERED' logs/*.log | wc -l)"
echo "Messages still buffered at end: $(grep 'Buffer size: [1-9]' logs/*.log | wc -l) processes with buffered msgs"
```

**Narration:**
> "Perfect! We can see that all messages have been delivered successfully. The algorithm correctly maintained causal ordering throughout the execution.
>
> Key observations:
> 1. Every message sent was also delivered
> 2. The final buffer is empty - all messages eventually delivered
> 3. Messages were buffered temporarily, but released when dependencies arrived
> 4. The system remained consistent throughout"

---

## Part 8: Code Walkthrough (1-2 minutes)

**Narration:**
> "Let's quickly examine the key algorithm implementation."

**On Screen:**
```bash
# Show the CanDeliver logic
less pkg/vectorclock/vectorclock.go
# Jump to CanDeliver function (around line 115)
```

**Narration (pointing at screen):**
> "Here's the core decision logic. When a message arrives with its vector information, we check: does this message have an entry for us? If it does, are all its dependencies satisfied?
>
> The key line: 'if entryForMe.Timestamp[j] > localTime[j]', we have an unsatisfied dependency.
>
> If all dependencies are satisfied, we can deliver immediately. Otherwise, we buffer and wait."

**Show:**
- CanDeliver function
- Dependency checking logic
- Return values (true/false with reason)

---

## Part 9: Algorithm Summary (1 minute)

**Narration:**
> "Let's summarize what we've demonstrated:
>
> **The SES Algorithm:**
> 1. Each process maintains a vector clock (one counter per process)
> 2. Before sending a message, we include our current vector clock and recent send history
> 3. Upon receiving a message, we check if its dependencies are satisfied
> 4. If dependencies are satisfied, deliver immediately and try to deliver any buffered messages
> 5. If not, buffer the message and try again later
>
> **Key Benefits:**
> - Ensures causal ordering of messages
> - Minimal network overhead (piggybacking only necessary info)
> - No central sequencer needed
> - Scales well to many processes
>
> **Our Results:**
> - Successfully coordinated 15 concurrent processes
> - Delivered over 20,000 messages with perfect causality
> - Buffer was fully processed - no message lost or stuck"

**On Screen:**
Show the README algorithm section

---

## Part 10: Conclusion (30 seconds)

**Narration:**
> "Thank you for watching this demonstration of the SES algorithm implementation. This project shows how distributed systems can maintain consistency and causality while minimizing communication overhead.
>
> The code is well-documented and available for review. For more details, please refer to the README.md, SETUP_GUIDE.md, and DESIGN.md files included in the project.
>
> Questions?"

**On Screen:**
- Show GitHub repository (if public)
- List all documentation files
- Final statistics screen

---

## Recording Tips

### Technical Setup
- Use OBS or ScreenFlow with 1080p resolution
- Frame rate: 30 fps minimum
- Microphone: Use external microphone for better quality
- Background: Clean, quiet space (minimal noise)

### During Recording
- Speak clearly at moderate pace
- Pause briefly after each section for editing
- If you make mistakes, pause and restart the section
- Highlight important parts by moving cursor over them
- Allow 2-3 seconds pause between major sections

### Post-Production Editing
- Cut out pauses and mistakes
- Add section titles/transitions
- Consider adding text overlays for key concepts
- Zoom in on code for clarity
- Final length: 5-10 minutes (tight, professional)

### Video Metadata
- Title: "SES Algorithm Implementation - Distributed Message Ordering"
- Description: Include link to GitHub/project files, brief summary
- Tags: distributed-systems, vector-clocks, message-ordering, Go

---

## Alternative Demo Scenarios

### Quick Demo (3 minutes)
- Skip detailed code walkthrough
- Focus on: Problem → Solution → Results
- Show only key log excerpts

### Extended Demo (15 minutes)
- Include detailed code walkthrough of all components
- Explain each algorithm step with examples
- Live coding: modify and recompile (e.g., show what happens without vector clocks)

### Interactive Demo
- Run processes interactively using the CLI
- Demonstrate the 's' (send), 'i' (info), 'b' (buffered), 'v' (vector) commands
- Show how you can monitor individual processes

---

## Expected Demo Output (For Reference)

When you run the demo, expect to see:
- Build completes in < 5 seconds
- Processes start and output "Process started successfully"
- SENT messages appear at ~100 per minute (configurable)
- RECEIVED messages appear shortly after
- DELIVERED messages appear after dependencies are satisfied
- Occasional BUFFERED messages (important for demonstration!)
- DELIVERING FROM BUFFER messages (shows algorithm working)
- Final statistics showing all messages delivered
- Total runtime: 45-75 seconds

---

## Troubleshooting for Recording

**Problem: Too many messages, screen is overwhelming**
- Solution: Grep to show specific events: `tail -f logs/process_0.log | grep DELIVERED`

**Problem: Processes finish too quickly**
- Solution: Modify config to increase `messages_per_process` or decrease `messages_per_minute`

**Problem: Port conflicts**
- Solution: Kill any stuck processes: `pkill -9 ses.exe`

**Problem: Audio issues**
- Solution: Re-record that section separately and edit in post

---

## Final Checklist Before Submission

- [ ] Video is 5-10 minutes long
- [ ] Audio is clear and professionally narrated
- [ ] All sections covered (intro, setup, execution, results, summary)
- [ ] Code is visible and readable
- [ ] Key concepts explained clearly
- [ ] Demo successfully shows buffering and delivery
- [ ] Final statistics verify correctness
- [ ] No background noise or distractions
- [ ] Video format: MP4, WebM, or YouTube link
- [ ] Subtitle/caption file included (optional but recommended)

---

**Script Version**: 1.0
**Estimated Recording Time**: 45 minutes (including multiple takes)
**Final Video Length**: 5-10 minutes
**Last Updated**: 2025-11-27
