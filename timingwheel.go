package timingwheel

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/RussellLuo/timingwheel/delayqueue"
)

// TimingWheel is an implementation of Hierarchical Timing Wheels.
type TimingWheel struct {
	tick      int64 // in milliseconds
	pointer   int64
	wheelSize int64
	layer     int64

	layers      int64
	interval    int64 // in milliseconds
	currentTime int64 // in milliseconds
	buckets     []*bucket
	queue       *delayqueue.DelayQueue

	// The higher-level overflow wheel.
	//
	// NOTE: This field may be updated and read concurrently, through Add().
	overflowWheel *TimingWheel // type: *TimingWheel
	// shuntflowWheel *TimingWheel

	exitC     chan struct{}
	waitGroup waitGroupWrapper
}

// NewTimingWheel creates an instance of TimingWheel with the given tick and wheelSize.
func NewTimingWheel(tick time.Duration, wheelSize int64, layers int64) *TimingWheel {
	tickSec := int64(tick / time.Second)
	if tickSec <= 0 {
		panic(errors.New("tick must be greater than or equal to 1s"))
	}

	now := time.Now()
	startSec := timeToSec(now.UTC())
	fmt.Printf("startSec: %d, %s\n", startSec, now.Format("2006-01-02 15:04:05.000"))

	return newTimingWheel(
		tickSec,
		wheelSize,
		1,
		layers,
		startSec,
		delayqueue.New(int(wheelSize)),
	)
}

// newTimingWheel is an internal helper function that really creates an instance of TimingWheel.
func newTimingWheel(tickSec int64, wheelSize int64, layer int64, layers int64, startSec int64, queue *delayqueue.DelayQueue) *TimingWheel {
	var buckets []*bucket
	if layer > layers {
		buckets = make([]*bucket, 0)
	} else {
		buckets = make([]*bucket, wheelSize)
		for i := range buckets {
			buckets[i] = newBucket()
		}
	}

	var overflowWheel *TimingWheel = nil
	if layer <= layers {
		overflowWheel = newTimingWheel(
			tickSec*wheelSize,
			wheelSize,
			layer+1,
			layers,
			startSec,
			delayqueue.New(int(wheelSize)),
		)
	}

	return &TimingWheel{
		tick:      tickSec,
		pointer:   0,
		wheelSize: wheelSize,
		layer:     layer,
		layers:    layers,
		// currentTime:   truncate(startMs, tickMs),
		currentTime:   startSec,
		interval:      tickSec * wheelSize,
		buckets:       buckets,
		queue:         queue,
		overflowWheel: overflowWheel,
		exitC:         make(chan struct{}),
	}
}

// func setcap(b []*bucket, cap int) []*bucket {
// 	s := make([]*bucket, cap)
// 	copy(s, b)
// 	return s
// }

func (tw *TimingWheel) Add(d time.Duration, f func()) *Timer {
	t := &Timer{
		id:         int64(d / time.Second),
		expiration: timeToSec(time.Now().UTC().Add(d)),
		task:       f,
	}
	tw.add(t)
	return t
}

// add inserts the timer t into the current timing wheel.
func (tw *TimingWheel) add(t *Timer) bool {
	// currentTime := atomic.LoadInt64(&tw.currentTime)
	currentTime := timeToSec(time.Now().UTC())
	// fmt.Printf("add layer[%d] tick[%d] interval[%d] expiration[%d] currentTime[%d]\n", tw.layer, tw.tick, tw.interval, t.expiration, currentTime)
	if (tw.layer == 1) && (t.expiration <= currentTime) {
		// Already expired
		t.task()
		return true
	} else if t.expiration < currentTime+tw.interval {
		// Put it into its own bucket
		virtualID := (t.expiration - currentTime) / tw.tick

		// fmt.Printf("task[%d] insert layer[%d] index: %d\n", t.expiration, tw.layer, (t.expiration-currentTime)/tw.tick)
		b := tw.buckets[(tw.pointer+virtualID)%tw.wheelSize]
		// b := tw.buckets[(tw.pointer+virtualID)%tw.wheelSize]
		b.Add(t)

		return true
	} else {
		if tw.layer <= tw.layers {
			// Out of the interval. Put it into the overflow wheel
			return tw.overflowWheel.add(t)
		} else {
			// Put it into its own bucket
			virtualID := t.expiration / tw.tick
			if cap(tw.buckets) < int(virtualID) {
				tw.buckets = setcap(tw.buckets, int(virtualID))
			}
			if tw.buckets[virtualID] == nil {
				tw.buckets[virtualID] = newBucket()
			}

			b := tw.buckets[virtualID]
			b.Add(t)
			return true
		}
	}
}

func (tw *TimingWheel) handler(t *Timer) {
	t.task()
}

func (tw *TimingWheel) reinsert(t *Timer) {
	tw.add(t)
}

func (tw *TimingWheel) load(reinsert func(*Timer)) {
	endPointer := (tw.pointer + 2) % tw.wheelSize
	index := (tw.pointer + 1) % tw.wheelSize
	for {
		if tw.pointer == endPointer {
			return
		}
		if tw.buckets[index].timers.Len() > 0 {
			tw.buckets[index].Load(reinsert)
		}
		time.Sleep(100 * time.Millisecond)
		// time.Sleep(time.Duration(tw.tick / tw.wheelSize))
	}
}

// Start starts the current timing wheel.
func (tw *TimingWheel) Start() {
	var (
		i int64 = 0
	)

	if tw.layer == 1 {
		for i = 0; i < tw.wheelSize; i++ {
			num := i
			go func(index int64) {
				for {
					if tw.buckets[index].timers.Len() > 0 {
						tw.buckets[index].Flush(tw.handler)
					}
					time.Sleep(100 * time.Millisecond)
				}
			}(num)
		}
	}

	start := atomic.LoadInt64(&tw.currentTime)
	tw.waitGroup.Wrap(func() {
		for {
			now := timeToSec(time.Now().UTC())
			if (now - start) >= tw.tick {
				tw.pointer = (tw.pointer + 1) % tw.wheelSize
				fmt.Printf("===TimingWheel[%d]=== pointer[%d] time[%s]\n", tw.layer, tw.pointer, time.Now().Format("2006-01-02 15:04:05.000"))
				start = now
				if tw.pointer == (tw.wheelSize - 1) {
					// fmt.Printf("timerWheel[layer%d] load from overflowWheel[layer%d]\n", tw.layer, tw.overflowWheel.layer)
					// start load
					// if tw.layer == 1 {
					// 	tw.Show()
					// }
					go tw.overflowWheel.load(tw.reinsert)
				}
			}
			// time.Sleep(20 * time.Millisecond)
			time.Sleep(time.Duration(tw.tick / 50))
		}
	})

	if tw.layer == tw.layers {
		return
	}
	tw.waitGroup.Wrap(func() {
		tw.overflowWheel.Start()
	})
	// go (*TimingWheel)(tw.overflowWheel).Start()
}

// Stop stops the current timing wheel.
//
// If there is any timer's task being running in its own goroutine, Stop does
// not wait for the task to complete before returning. If the caller needs to
// know whether the task is completed, it must coordinate with the task explicitly.
func (tw *TimingWheel) Stop() {
	close(tw.exitC)
	tw.waitGroup.Wait()
}

func (tw *TimingWheel) Show() {
	fmt.Printf("layer: %d\n", tw.layer)
	for i := 0; i < int(tw.wheelSize); i++ {
		len := tw.buckets[i].timers.Len()
		if len != 0 {
			fmt.Printf("i: %d, len: %d\n", i, len)
		}
	}
	fmt.Printf("layer: %d\n", tw.overflowWheel.layer)
	for i := 0; i < int(tw.overflowWheel.wheelSize); i++ {
		len := tw.overflowWheel.buckets[i].timers.Len()
		if len != 0 {
			fmt.Printf("i: %d, len: %d\n", i, len)
		}
	}
	fmt.Printf("layer: %d\n", tw.overflowWheel.overflowWheel.layer)
	for i := 0; i < int(tw.overflowWheel.overflowWheel.wheelSize); i++ {
		len := tw.overflowWheel.overflowWheel.buckets[i].timers.Len()
		if len != 0 {
			fmt.Printf("i: %d, len: %d\n", i, len)
		}
	}
}
