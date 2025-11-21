package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/NationalWind/ses-project/pkg/process"
)

type Config struct {
	NumProcesses       int             `json:"num_processes"`
	MessagesPerProcess int             `json:"messages_per_process"`
	MessagesPerMinute  int             `json:"messages_per_minute"`
	Processes          []ProcessConfig `json:"processes"`
}

type ProcessConfig struct {
	ID      int    `json:"id"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

func main() {
	config, err := loadConfig("config/config.json")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll("logs", 0755); err != nil {
		fmt.Printf("Error creating logs directory: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Println("Missing process ID argument")
		os.Exit(1)
	}

	processID, err := strconv.Atoi(os.Args[1])
	if err != nil || processID < 0 || processID >= config.NumProcesses {
		fmt.Println("Invalid process ID")
		os.Exit(1)
	}

	autoSend := false
	if len(os.Args) >= 3 && os.Args[2] == "send" {
		autoSend = true
	}

	// Build peers map and get own config
	peers := make(map[int]string)
	var myConfig ProcessConfig
	for _, pc := range config.Processes {
		if pc.ID == processID {
			myConfig = pc
		} else {
			peers[pc.ID] = fmt.Sprintf("%s:%d", pc.Address, pc.Port)
		}
	}

	// Create process
	p, err := process.NewProcess(
		processID,
		myConfig.Address,
		myConfig.Port,
		config.NumProcesses,
		peers,
	)
	if err != nil {
		fmt.Printf("Error creating process: %v\n", err)
		os.Exit(1)
	}
	defer p.Close()

	if err := p.Start(); err != nil {
		fmt.Printf("Error starting process: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[P%d] Process started successfully!\n", processID)

	// Nếu autoSend = true thì gửi message và exit luôn
	if autoSend {
		fmt.Printf("[P%d] Auto sending messages...\n", processID)
		p.SendMessages(config.MessagesPerProcess, config.MessagesPerMinute)
		return
	}

	// Interactive mode nếu không autoSend
	fmt.Println("\nCommands:")
	fmt.Println("  's' - Start sending messages")
	fmt.Println("  'i' - Show statistics")
	fmt.Println("  'b' - Show buffered messages")
	fmt.Println("  'v' - Show vector clock")
	fmt.Println("  'q' - Quit")
	fmt.Print("\n> ")

	// Interactive loop
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		cmd := scanner.Text()
		switch cmd {
		case "s":
			go p.SendMessages(config.MessagesPerProcess, config.MessagesPerMinute)
		case "i":
			printStats(p)
		case "b":
			printBuffered(p)
		case "v":
			printVectorClock(p)
		case "q":
			fmt.Println("Shutting down...")
			return
		default:
			fmt.Println("Unknown command")
		}
		fmt.Print("\n> ")
	}
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// Hàm helper để in stats
func printStats(p *process.Process) {
	stats := p.GetStats()
	fmt.Println("\n=== Process Statistics ===")
	fmt.Printf("Process ID: %d\n", stats["id"])
	fmt.Printf("Vector Clock: %v\n", stats["vector_clock"])
	fmt.Printf("Delivered Messages: %d\n", stats["delivered_count"])
	fmt.Printf("Buffered Messages: %d\n", stats["buffered_count"])
	fmt.Println("\nSent Messages:")
	for id, count := range stats["sent_messages"].(map[int]int) {
		fmt.Printf("  To P%d: %d\n", id, count)
	}
	fmt.Println("Received Messages:")
	for id, count := range stats["received_messages"].(map[int]int) {
		fmt.Printf("  From P%d: %d\n", id, count)
	}
}

func printBuffered(p *process.Process) {
	stats := p.GetStats()
	fmt.Printf("\nBuffered Messages: %d\n", stats["buffered_count"])
}

func printVectorClock(p *process.Process) {
	stats := p.GetStats()
	fmt.Printf("\nVector Clock: %v\n", stats["vector_clock"])
}
