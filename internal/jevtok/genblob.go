//go:build ignore

// Generator for the embedded Jev rank tables.
//
// Provenance: Oh My Pi commit cdd36ca (MIT; see NOTICE).
//
//	crates/pi-natives/data/jev_base.bin.zst   sha256 557ff3d99edf090f6aa9bb8679ffeb63f3a372470ba89b71a7255d0f2148dfbb
//	crates/pi-natives/data/jev_whole.bin.zst  sha256 ea9a1e545a32c52b5a26e623d504c8da51e6417d7ff9c3f92287db23c1dc6518
//
// Those blobs are zstd-19 of a UTOK1 rank table (magic "UTOK1\n", u32le entry
// count 199998, then varint length + token bytes; rank = entry index; empty
// entries are dead slots). Runtime uses compress/gzip, so this tool
// decompresses with the zstd CLI and writes data/jev_*.bin.gz.
//
//	go run genblob.go /path/to/jev_base.bin.zst /path/to/jev_whole.bin.zst
//
// With no args it fetches the two blobs from
// https://raw.githubusercontent.com/can1357/oh-my-pi/cdd36ca/crates/pi-natives/data/
package main

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const rawBase = "https://raw.githubusercontent.com/can1357/oh-my-pi/cdd36ca/crates/pi-natives/data/"

func main() {
	base, whole := "jev_base.bin.zst", "jev_whole.bin.zst"
	if len(os.Args) == 3 {
		base, whole = os.Args[1], os.Args[2]
	} else if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: go run genblob.go [jev_base.bin.zst jev_whole.bin.zst]")
		os.Exit(2)
	} else {
		base, whole = fetch(base), fetch(whole)
	}
	dir := filepath.Join(filepath.Dir(os.Args[0]), "data")
	// go run puts the binary in a temp dir; write next to this source file.
	if src, err := os.Getwd(); err == nil {
		dir = filepath.Join(src, "data")
	}
	writeGZ(base, filepath.Join(dir, "jev_base.bin.gz"))
	writeGZ(whole, filepath.Join(dir, "jev_whole.bin.gz"))
}

func fetch(name string) string {
	resp, err := http.Get(rawBase + name)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		panic(resp.Status)
	}
	f, err := os.CreateTemp("", name)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		panic(err)
	}
	f.Close()
	return f.Name()
}

func writeGZ(zst, dst string) {
	cmd := exec.Command("zstd", "-d", "-c", zst)
	raw, err := cmd.Output()
	if err != nil {
		panic(fmt.Errorf("zstd %s: %w", zst, err))
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		panic(err)
	}
	f, err := os.Create(dst)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	if _, err := w.Write(raw); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	sum := sha256.Sum256(raw)
	fmt.Printf("%s: %d raw sha256 %x -> %s\n", zst, len(raw), sum, dst)
}
