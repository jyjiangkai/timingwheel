package timingwheel_test

// import (
// 	"fmt"
// 	"testing"
// 	"time"

// 	"github.com/RussellLuo/timingwheel"
// )

// func timeToMs(t time.Time) int64 {
// 	fmt.Println(t.UnixNano())
// 	fmt.Println(int64(time.Millisecond))
// 	return t.UnixNano() / int64(time.Millisecond)
// }

// func TestTimingWheel_MyFunc(t *testing.T) {
// 	fmt.Println(timeToMs(time.Now().UTC()))
// }
// func TestTimingWheel_AfterFunc(t *testing.T) {
// 	tw := timingwheel.NewTimingWheel(time.Millisecond, 10, 4)
// 	tw.Start()
// 	defer tw.Stop()

// 	durations := []time.Duration{
// 		5 * time.Millisecond,
// 		15 * time.Millisecond,
// 		125 * time.Millisecond,
// 		1250 * time.Millisecond,
// 	}
// 	for _, d := range durations {
// 		t.Run("", func(t *testing.T) {
// 			exitC := make(chan time.Time)

// 			fmt.Printf("====== %s\n", time.Now().Format("2006-01-02 15:04:05.000"))
// 			start := time.Now().UTC()
// 			tw.AfterFunc(d, func() {
// 				fmt.Printf("====== %s\n", time.Now().Format("2006-01-02 15:04:05.000"))
// 				exitC <- time.Now().UTC()
// 			})

// 			got := (<-exitC).Truncate(time.Millisecond)
// 			min := start.Add(d).Truncate(time.Millisecond)

// 			err := 5 * time.Millisecond
// 			if got.Before(min) || got.After(min.Add(err)) {
// 				t.Errorf("Timer(%s) expiration: want [%s, %s], got %s", d, min, min.Add(err), got)
// 			}
// 		})
// 	}
// }

// type scheduler struct {
// 	intervals []time.Duration
// 	current   int
// }

// func (s *scheduler) Next(prev time.Time) time.Time {
// 	if s.current >= len(s.intervals) {
// 		return time.Time{}
// 	}
// 	next := prev.Add(s.intervals[s.current])
// 	s.current += 1
// 	return next
// }

// func TestTimingWheel_ScheduleFunc(t *testing.T) {
// 	tw := timingwheel.NewTimingWheel(time.Millisecond, 20, 4)
// 	tw.Start()
// 	defer tw.Stop()

// 	s := &scheduler{intervals: []time.Duration{
// 		1 * time.Millisecond,   // start + 1ms
// 		4 * time.Millisecond,   // start + 5ms
// 		5 * time.Millisecond,   // start + 10ms
// 		40 * time.Millisecond,  // start + 50ms
// 		50 * time.Millisecond,  // start + 100ms
// 		400 * time.Millisecond, // start + 500ms
// 		500 * time.Millisecond, // start + 1s
// 	}}

// 	exitC := make(chan time.Time, len(s.intervals))

// 	start := time.Now().UTC()
// 	tw.ScheduleFunc(s, func() {
// 		exitC <- time.Now().UTC()
// 	})

// 	accum := time.Duration(0)
// 	for _, d := range s.intervals {
// 		got := (<-exitC).Truncate(time.Millisecond)
// 		accum += d
// 		min := start.Add(accum).Truncate(time.Millisecond)

// 		err := 5 * time.Millisecond
// 		if got.Before(min) || got.After(min.Add(err)) {
// 			t.Errorf("Timer(%s) expiration: want [%s, %s], got %s", accum, min, min.Add(err), got)
// 		}
// 	}
// }
