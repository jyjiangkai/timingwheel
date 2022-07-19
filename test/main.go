package main

import (
	"fmt"
	"time"

	"github.com/RussellLuo/timingwheel"
)

func TestTimingWheel() {
	tw := timingwheel.NewTimingWheel(time.Second, 10, 4)
	tw.Start()
	defer tw.Stop()

	handler := func() {
		fmt.Printf("===task=== %s\n", time.Now().Format("2006-01-02 15:04:05.000"))
	}

	tw.Add(2*time.Second, handler)
	tw.Add(5*time.Second, handler)
	tw.Add(10*time.Second, handler)
	tw.Add(20*time.Second, handler)
	tw.Add(30*time.Second, handler)
	tw.Add(100*time.Second, handler)
	tw.Add(110*time.Second, handler)

	for {
	}
}

func main() {
	TestTimingWheel()
}
