package main

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
)

type Block struct {
	last     *Block
	username string
	param    string
	value    float64
	id       int
	hash     []byte
}

func (blk *Block) Append(username string, parameter string, value float64) (*Block, error) {
	sol := &Block{
		last:     blk,
		username: username,
		param:    parameter,
		value:    value,
		id:       blk.id + 1,
	}
	h := sha256.New()
	h.Write([]byte(sol.username))
	h.Write([]byte(sol.param))
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], math.Float64bits(sol.value))
	h.Write(buf[:])
	if sol.last != nil {
		h.Write(sol.last.hash)
	}
	sol.hash = h.Sum(nil)
	return sol, nil
}

func (blk *Block) getBlock(i int) (*Block, error) {
	if blk.id < i || i < 0 {
		return nil, errors.New(fmt.Sprint("No such id!", i))
	}
	for blk.id != i {
		blk = blk.last
	}
	return blk, nil
}

func (blk *Block) getState() (map[string]map[string]float64, error) {
	sol := make(map[string]map[string]float64)
	curr := blk
	for curr != nil && curr.last != nil {
		val, ok := sol[curr.username]
		if !ok {
			sol[curr.username] = make(map[string]float64)
			val = sol[curr.username]
		}
		if _, ok := val[curr.param]; !ok {
			val[curr.param] = curr.value
		}
		curr = curr.last
	}
	return sol, nil
}

var LastBlock *Block = &Block{id: 0}
var blockMuter sync.Mutex

func PeekLast() *Block {
	return LastBlock
}

func Append(username string, parameter string, value float64) (*Block, error) {
	blockMuter.Lock()
	defer blockMuter.Unlock()
	sol, err := LastBlock.Append(username, parameter, value)
	if err != nil {
		return nil, err
	}
	LastBlock = sol
	return sol, nil
}
