package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var jsonCommandFile string
var delayNext = 5 * time.Second
var commands [][]string
var sig = make(chan os.Signal, 1)

func init() {
	flag.DurationVar(&delayNext, "delay", delayNext, "delay between starting one command and the next")
	flag.StringVar(&jsonCommandFile, "json", jsonCommandFile, "the json [\"\",\"\"][\"\",\"\"] file specifying commands to run")
}

func main() {
	var fp *os.File
	var err error

	flag.Parse()

	if jsonCommandFile == "-" {
		fp = os.Stdin
	} else {
		fp, err = os.Open(jsonCommandFile)
		if err != nil {
			log.Fatal(err)
		}
	}
	dec := json.NewDecoder(fp)
	for {
		var command []string
		err := dec.Decode(&command)
		if command != nil {
			if len(command) > 0 {
				commands = append(commands, command)
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
			}
			break
		}
	}

	signal.Notify(sig, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGSTOP)

	var ctx, cancel = context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-sig:
				cancel()
				fmt.Println("cancelling")
			}
		}
	}()

	var wg sync.WaitGroup
	for _, cmd := range commands {
		wg.Add(1)
		go func(c []string) {
			fmt.Println("executing", c)
			var runner *exec.Cmd
			if len(c) > 1 {
				runner = exec.CommandContext(ctx, c[0], c[1:]...)
			} else {
				runner = exec.CommandContext(ctx, c[0])
			}
			runner.Stderr = os.Stderr
			runner.Stdout = os.Stdout
			if err := runner.Run(); err != nil {
				log.Println(err.Error())
			}
			wg.Done()
		}(cmd)
		time.Sleep(delayNext)
	}
	wg.Wait()
}
