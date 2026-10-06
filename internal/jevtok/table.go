package jevtok

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"slices"
	"sync"
)

//go:embed data/jev_base.bin.gz data/jev_whole.bin.gz
var blobs embed.FS

const (
	window       = 512
	rankMissing  = math.MaxUint32
	o200kEntries = 199998
)

// packKey holds a token of 1 or 3..=15 bytes. Map keys must not allocate.
type packKey struct {
	lo, hi uint64
	n      uint8
}

func packOf(b []byte) packKey {
	var k packKey
	k.n = uint8(len(b))
	for i, c := range b {
		if i < 8 {
			k.lo |= uint64(c) << (uint(i) * 8)
		} else {
			k.hi |= uint64(c) << (uint(i-8) * 8)
		}
	}
	return k
}

// rankTable is an o200k-rank subset. Empty UTOK1 slots are absent, so merges
// cannot produce them. pairs is the 2-byte hot path; short/long cover the rest.
type rankTable struct {
	pairs [65536]uint32
	short map[packKey]uint32
	long  map[string]uint32
}

func (t *rankTable) rank(b []byte) (uint32, bool) {
	if len(b) == 2 {
		r := t.pairs[uint32(b[0])<<8|uint32(b[1])]
		return r, r != rankMissing
	}
	if len(b) == 0 || len(b) > 15 {
		r, ok := t.long[string(b)]
		return r, ok
	}
	r, ok := t.short[packOf(b)]
	return r, ok
}

func (t *rankTable) has(b []byte) bool {
	_, ok := t.rank(b)
	return ok
}

// countMerged is tiktoken's byte_pair_merge counted, without the whole-piece
// short-circuit. A table hit the merges cannot reach stays split.
func (t *rankTable) countMerged(piece []byte) int {
	if len(piece) == 0 {
		return 0
	}
	n := 0
	t.merge(piece, func(_, _ int) { n++ })
	return n
}

func (t *rankTable) merge(piece []byte, emit func(start, end int)) {
	// parts[k] = (start, rank of merging part k with part k+1). Two sentinels
	// keep parts[k+3] in range when recomputing the rank after a merge.
	parts := make([][2]int, 0, len(piece)+1)
	minRank, minAt := rankMissing, -1
	for i := range len(piece) - 1 {
		rank := rankMissing
		if r, ok := t.rank(piece[i : i+2]); ok {
			rank = int(r)
		}
		if rank < minRank {
			minRank, minAt = rank, i
		}
		parts = append(parts, [2]int{i, int(rank)})
	}
	parts = append(parts, [2]int{len(piece) - 1, int(rankMissing)}, [2]int{len(piece), int(rankMissing)})

	getRank := func(k int) uint32 {
		if k+3 < len(parts) {
			if r, ok := t.rank(piece[parts[k][0]:parts[k+3][0]]); ok {
				return r
			}
		}
		return rankMissing
	}

	for minRank != rankMissing {
		i := minAt
		if i > 0 {
			parts[i-1][1] = int(getRank(i - 1))
		}
		parts[i][1] = int(getRank(i))
		parts = slices.Delete(parts, i+1, i+2)

		minRank, minAt = rankMissing, -1
		for k, p := range parts[:len(parts)-1] {
			if p[1] < minRank {
				minRank, minAt = p[1], k
			}
		}
	}
	for i := range len(parts) - 1 {
		emit(parts[i][0], parts[i+1][0])
	}
}

func parseUTOK(gz []byte) *rankTable {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		panic(fmt.Errorf("jevtok: gzip: %w", err))
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		panic(fmt.Errorf("jevtok: gzip read: %w", err))
	}
	if len(raw) < 10 || string(raw[:6]) != "UTOK1\n" {
		panic("jevtok: bad UTOK1 magic")
	}
	n := binary.LittleEndian.Uint32(raw[6:10])
	if n != o200kEntries {
		panic(fmt.Errorf("jevtok: entry count %d", n))
	}
	t := &rankTable{
		short: make(map[packKey]uint32, 1<<16),
		long:  make(map[string]uint32),
	}
	for i := range t.pairs {
		t.pairs[i] = rankMissing
	}
	p := raw[10:]
	for rank := uint32(0); rank < n; rank++ {
		length, shift := 0, uint(0)
		for {
			if len(p) == 0 {
				panic("jevtok: truncated varint")
			}
			b := p[0]
			p = p[1:]
			length |= int(b&0x7f) << shift
			if b < 0x80 {
				break
			}
			shift += 7
		}
		if length == 0 {
			continue
		}
		if len(p) < length {
			panic("jevtok: truncated token")
		}
		key := p[:length]
		p = p[length:]
		switch {
		case length == 2:
			t.pairs[uint32(key[0])<<8|uint32(key[1])] = rank
		case length <= 15:
			t.short[packOf(key)] = rank
		default:
			t.long[string(key)] = rank
		}
	}
	if len(p) != 0 {
		panic("jevtok: trailing UTOK1 bytes")
	}
	return t
}

var (
	tablesOnce sync.Once
	baseTable  *rankTable
	wholeTable *rankTable
)

func tables() (*rankTable, *rankTable) {
	tablesOnce.Do(func() {
		base, err := blobs.ReadFile("data/jev_base.bin.gz")
		if err != nil {
			panic(err)
		}
		whole, err := blobs.ReadFile("data/jev_whole.bin.gz")
		if err != nil {
			panic(err)
		}
		baseTable = parseUTOK(base)
		wholeTable = parseUTOK(whole)
	})
	return baseTable, wholeTable
}
