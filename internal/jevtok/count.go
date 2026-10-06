package jevtok

import "golang.org/x/text/unicode/norm"

// Count returns Jev 1.13 input tokens for text as state content.
//
// NFC, then the Qwen3.5 split. A piece in the whole-word table costs 1;
// anything else is merged in independent 512-byte windows on the base
// table (o200k ranks, non-base slots empty, no whole-piece short-circuit).
func Count(text string) int {
	if text != "" && !norm.NFC.IsNormalString(text) {
		text = norm.NFC.String(text)
	}
	base, whole := tables()
	n := 0
	forEachPiece(text, func(piece string) {
		b := []byte(piece)
		if whole.has(b) {
			n++
			return
		}
		for len(b) > 0 {
			w := b
			if len(w) > window {
				w = w[:window]
			}
			n += base.countMerged(w)
			b = b[len(w):]
		}
	})
	return n
}
