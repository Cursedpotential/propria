// Byline: Claude Code · Sonnet · 2026-10-02
//
// Scrambled-bytes detection for repair.find_other_version.
//
// 2026-10-02: five casevault HTML files (Facebook account_activity, your_friends, your_post_audiences,
// 30.html, a Takeout MyActivity.html) were found to be scrambled bytes: entropy 7.99 bits per byte over
// the whole file, no format marker, no compression, and a catalog sha1 that equals the scrambled bytes. The
// all-zero check (a zero-filled husk) cannot see them. A head is judged scrambled only when ALL of these
// hold: it is long enough to judge, it carries none of the format markers a real file of any type starts
// with, it is not clean text, and its entropy is that of random data. A legitimately compressed file
// (jpeg, mp4, zip, ...) always starts with its marker, so it never reaches the entropy test.
package activities

import (
	"bytes"
	"math"
	"unicode/utf8"
)

const (
	// scrambleHeadBytes is how much of a head the judgement reads.
	scrambleHeadBytes = 4096
	// scrambleMinHead is the shortest head that proves anything about entropy.
	scrambleMinHead = 1024
	// scrambleEntropy is the bits per byte at or above which a marker-less, non-text head counts as
	// random. 4 KiB of random data measures about 7.95; ordinary binary structure stays well below.
	scrambleEntropy = 7.5
)

const rejectScrambled = "scrambled (high-entropy head with no format marker)"

// formatMarkers are the leading-byte signatures of the file types in the casevault. A head matching any
// of them is a real file of that type, whatever its extension says.
var formatMarkers = []func(b []byte) bool{
	func(b []byte) bool { return bytes.HasPrefix(b, []byte{0xff, 0xd8, 0xff}) },    // jpeg
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")) }, // png
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("GIF8")) },              // gif
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("RIFF")) },              // webp, wav, avi
	func(b []byte) bool { return len(b) >= 8 && isoBMFFBox(b[4:8]) },               // mp4, mov, heic
	func(b []byte) bool {
		return bytes.HasPrefix(b, []byte("ID3")) || (len(b) > 1 && b[0] == 0xff && b[1]&0xe0 == 0xe0)
	}, // mp3
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("%PDF-")) },                                      // pdf
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("PK")) },                                         // zip family
	func(b []byte) bool { return bytes.HasPrefix(b, []byte{0x1f, 0x8b}) },                                   // gzip
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("7z\xbc\xaf\x27\x1c")) },                         // 7z
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("Rar!")) },                                       // rar
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("\xfd7zXZ\x00")) },                               // xz
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("BZh")) },                                        // bzip2
	func(b []byte) bool { return bytes.HasPrefix(b, []byte{0x28, 0xb5, 0x2f, 0xfd}) },                       // zstd
	func(b []byte) bool { return len(b) >= 262 && bytes.Equal(b[257:262], []byte("ustar")) },                // tar
	func(b []byte) bool { return bytes.HasPrefix(b, []byte{0xca, 0xfe, 0xba, 0xbe}) },                       // java class
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("SQLite format 3")) },                            // sqlite
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("MZ")) },                                         // exe
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("\x7fELF")) },                                    // elf
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("OggS")) || bytes.HasPrefix(b, []byte("fLaC")) }, // ogg, flac
	func(b []byte) bool { return bytes.HasPrefix(b, []byte{0x1a, 0x45, 0xdf, 0xa3}) },                       // mkv, webm
	func(b []byte) bool {
		return bytes.HasPrefix(b, []byte("II*\x00")) || bytes.HasPrefix(b, []byte("MM\x00*"))
	}, // tiff
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("BM")) },                   // bmp
	func(b []byte) bool { return bytes.HasPrefix(b, []byte{0x00, 0x00, 0x01, 0x00}) }, // ico
	func(b []byte) bool { // fonts
		return bytes.HasPrefix(b, []byte{0x00, 0x01, 0x00, 0x00}) || bytes.HasPrefix(b, []byte("OTTO")) ||
			bytes.HasPrefix(b, []byte("wOFF")) || bytes.HasPrefix(b, []byte("wOF2")) || bytes.HasPrefix(b, []byte("ttcf"))
	},
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1")) }, // ole2 (legacy office)
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("{\\rtf")) },                           // rtf
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("8BPS")) },                             // psd
	func(b []byte) bool { return bytes.HasPrefix(b, []byte("PAR1")) || bytes.HasPrefix(b, []byte("ARROW1")) },
	func(b []byte) bool {
		return bytes.HasPrefix(b, []byte("-----BEGIN")) || bytes.HasPrefix(b, []byte{0x99, 0x01}) || bytes.HasPrefix(b, []byte{0x89, 0x01})
	}, // pgp
}

func isoBMFFBox(tag []byte) bool {
	switch string(tag) {
	case "ftyp", "moov", "mdat", "free", "wide", "skip", "styp":
		return true
	}
	return false
}

func hasFormatMarker(head []byte) bool {
	for _, marker := range formatMarkers {
		if marker(head) {
			return true
		}
	}
	return false
}

// cleanText reports a head that is text: a text byte-order mark, or valid UTF-8 (a character cut at the end
// of the sample is tolerated) that is almost all printable with no NUL byte.
func cleanText(head []byte) bool {
	if bytes.HasPrefix(head, []byte{0xef, 0xbb, 0xbf}) || bytes.HasPrefix(head, []byte{0xff, 0xfe}) || bytes.HasPrefix(head, []byte{0xfe, 0xff}) {
		return true
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return false
	}
	valid := head
	for cut := 0; cut < 4 && len(valid) > 0 && !utf8.Valid(valid); cut++ {
		valid = valid[:len(valid)-1]
	}
	if !utf8.Valid(valid) {
		return false
	}
	printable := 0
	for _, b := range head {
		if (b >= 32 && b < 127) || b == '\t' || b == '\n' || b == '\r' || b >= 0xc2 {
			printable++
		}
	}
	return float64(printable)/float64(len(head)) >= 0.92
}

func headEntropy(head []byte) float64 {
	if len(head) == 0 {
		return 0
	}
	var counts [256]int
	for _, b := range head {
		counts[b]++
	}
	n := float64(len(head))
	entropy := 0.0
	for _, c := range counts {
		if c == 0 {
			continue
		}
		p := float64(c) / n
		entropy -= p * math.Log2(p)
	}
	return entropy
}

// looksScrambled judges the first bytes of the object called name. The name is not an input to the
// verdict (a scrambled file keeps its name); it only appears in the signature for the caller's logs.
func looksScrambled(_ string, head []byte) bool {
	if len(head) > scrambleHeadBytes {
		head = head[:scrambleHeadBytes]
	}
	if len(head) < scrambleMinHead {
		return false
	}
	if hasFormatMarker(head) || cleanText(head) {
		return false
	}
	return headEntropy(head) >= scrambleEntropy
}
