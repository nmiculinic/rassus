package main

import (
	"time"
	"math/rand"
	"math"
	"log"
	"sync"
)

var (
	StartTime time.Time
	jitter = 0.2 * 2*(rand.Float64() - 0.5)
	correction float64 = 0
	scalarMutex = sync.Mutex{}
	vectorMutex = sync.Mutex{}

)
func init()  {
	StartTime = time.Now()
	log.Print("Jitter is ", jitter)
}

func emulatedSystemClock() time.Time {
	diff := time.Now().Sub(StartTime)
	delta := correction + diff.Seconds() * math.Pow(1+jitter, diff.Seconds()/1000.0)
	return StartTime.Add(time.Duration(delta * float64(time.Second)))
}

type ScalarTimestamp struct{
	Time time.Time `json:"time"`
}


func UpdateScalar(other *ScalarTimestamp) {
	scalarMutex.Lock()
	defer scalarMutex.Unlock()
	correction += other.Time.Sub(emulatedSystemClock()).Seconds()
}

func NowScalar() ScalarTimestamp {
	return ScalarTimestamp{
		emulatedSystemClock(),
	}
}

type VectorTimestamp struct {
	Time map[string]int64 `json:"time"`
}

func Max(x, y int64) int64 {
    if x > y {
        return x
    }
    return y
}

func (curr *VectorTimestamp) UpdateVector(other *VectorTimestamp, me string) VectorTimestamp {
	sol := VectorTimestamp{
		make(map[string]int64),
	}
	for k := range other.Time {
		sol.Time[k] = Max(curr.Time[k], other.Time[k])
	}
	return sol
}
