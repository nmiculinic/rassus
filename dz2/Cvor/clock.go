package main

import (
	"log"
	"math"
	"math/rand"
	"time"
)

type Timestamp interface {
	Now() Timestamp
	Update(timestamp *Timestamp) Timestamp
}

type ScalarTimestamp struct {
	Time       time.Time `json:"time"`
	startTime  time.Time
	jitter     float64
	correction float64
}

func (clock *ScalarTimestamp) Now() ScalarTimestamp {
	diff := time.Since(clock.startTime)
	delta := clock.correction + diff.Seconds()*math.Pow(1+clock.jitter, diff.Seconds()/1000.0)
	return ScalarTimestamp{
		Time:       clock.startTime.Add(time.Duration(delta * float64(time.Second))),
		startTime:  clock.startTime,
		jitter:     clock.jitter,
		correction: clock.correction,
	}
}

func (clock *ScalarTimestamp) Update(other *ScalarTimestamp, me string) ScalarTimestamp {
	sol := ScalarTimestamp{
		startTime:  clock.startTime,
		jitter:     clock.jitter,
		correction: clock.correction + other.Time.Sub(clock.Now().Time).Seconds(),
	}
	return sol.Now()
}

func NewScalar() ScalarTimestamp {
	sol := ScalarTimestamp{
		startTime: time.Now(),
		jitter:    0.2 * 2 * (rand.Float64() - 0.5),
	}
	return sol.Now()
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

func (curr *VectorTimestamp) Now() VectorTimestamp {
	return curr.Update(nil, "")
}

func (curr *VectorTimestamp) Update(other *VectorTimestamp, me string) VectorTimestamp {
	sol := make(map[string]int64)
	if other != nil {
		log.Println(curr.Time)
		log.Println(other.Time)
		for k := range other.Time {
			sol[k] = Max(curr.Time[k], other.Time[k])
		}
	} else {
		for k, v := range curr.Time {
			sol[k] = v
		}
	}
	if me != "" {
		sol[me]++
	}
	//if other != nil {
	//	log.Println(curr.Time)
	//	log.Println(other.Time)
	//	log.Println(sol)
	//}
	return VectorTimestamp{
		sol,
	}
}

func NewVectorTimestamp(agents []string) VectorTimestamp {
	m := make(map[string]int64)
	for _, agent := range agents {
		m[agent] = 0
		log.Print("has", agent)
	}
	log.Print(m)
	return VectorTimestamp{
		m,
	}
}
