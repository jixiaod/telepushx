package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Function to ping targets and log results
func pingAndLog(targets []string) {
	for i := 0; i < 30; i++ {
		for _, target := range targets {
			cmd := exec.Command("ping", "-c", "1", target)
			err := cmd.Run()
			if err != nil {
				fmt.Printf("Ping to %s failed: %v\n", target, err)
			} else {
				fmt.Printf("Ping to %s succeeded\n", target)
			}
		}
		time.Sleep(1 * time.Second) // Wait 1 second before the next ping
	}

}

func main() {
	logFilePath := "../logs/info." + time.Now().Format("2006-01-02") + ".log"
	fmt.Println("Current log file path:", logFilePath)
	timeoutKeyword := "Gateway Timeout"
	pingTargets := []string{"api.telegram.org", "www.google.com"}

	// Create a channel to signal when to stop monitoring
	stopChan := make(chan struct{})

	// Function to monitor the log file for new entries
	go func() {
		file, err := os.Open(logFilePath)
		if err != nil {
			fmt.Printf("Error opening log file: %v\n", err)
			return
		}
		defer file.Close()

		// Seek to the end of the file
		_, err = file.Seek(0, os.SEEK_END)
		if err != nil {
			fmt.Printf("Error seeking log file: %v\n", err)
			return
		}

		reader := bufio.NewReader(file)
		for {
			select {
			case <-stopChan:
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					time.Sleep(1 * time.Second) // Wait before trying to read again
					continue
				}
				if strings.Contains(line, timeoutKeyword) {
					// If "Gateway Timeout" is found, start pinging
					go pingAndLog(pingTargets)
				}
			}
		}
	}()

	// Cleanup function to stop monitoring
	defer close(stopChan)

	// Block forever
	select {}
}
