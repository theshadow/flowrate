// Copyright © 2019 Xander Guzman <xander.guzman@xanderguzman.com>
// A POC for testing the flow rate idea.

// Question: Can I build a system that uses a PID Controller
// (https://en.wikipedia.org/wiki/PID_controller) to manage the output flow rate.

package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"time"
)

type SyncCtrl struct {
	wg  *sync.WaitGroup
	ctx context.Context
}

// does things
// cvCh Control Value regulates the time between generation
func generator(ctrl *SyncCtrl, cvCh <-chan int64) <-chan int {
	ch := make(chan int)

	ctrl.wg.Add(1)

	go func() {
		defer fmt.Println("generator terminated")
		defer close(ch)
		defer ctrl.wg.Done()

		var cv int64

		for {
			select {
			case _ = <-ctrl.ctx.Done():
				return
			case cv = <-cvCh:
			default:
				v := rand.Intn(100)
				ch <- v

				// if we have a negative value
				if cv > 0 {
					time.Sleep(time.Duration(cv))
				}
			}
		}
	}()
	return ch
}

// returns two channels: samples, out
// out is where the messages that are consumed are piped through
func sampler(ctrl *SyncCtrl, in <-chan int, d time.Duration) (<-chan int, <-chan int) {
	samples := make(chan int)
	out := make(chan int)

	ctrl.wg.Add(1)

	go func() {
		defer fmt.Println("sampler terminated")
		defer close(out)
		defer close(samples)
		defer ctrl.wg.Done()

		t := time.NewTicker(d)
		defer t.Stop()

		var c int

		for {
			select {
			case _ = <-t.C:
				samples <- c
				c = 0
			case _ = <-ctrl.ctx.Done():
				return
			case v := <-in:
				out <- v
				c++
			}
		}
	}()

	return samples, out
}

func writer(in <-chan int, w io.Writer) {
	go func() {
		for v := range in {
			_, err := w.Write([]byte(fmt.Sprintf("value: %v\n", v)))
			if err != nil {
				log.Printf("unable to write value to writer: %v", err)
			}
		}
	}()
}

func main() {
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())

	sc := SyncCtrl{
		wg:  &wg,
		ctx: ctx,
	}

	cvCh := make(chan int64)

	ints := generator(&sc, cvCh)
	pvs, out := sampler(&sc, ints, time.Second*1)

	writer(out, ioutil.Discard)

	sp := 1000

	go func() {
		for s := range pvs {
			fmt.Printf("sample: %v\n", s)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	go func() {
		<-sig
		close(cvCh)
		cancel()
	}()

	wg.Wait()
}
