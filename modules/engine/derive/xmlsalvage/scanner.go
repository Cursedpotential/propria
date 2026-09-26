// Package xmlsalvage finds how much of a cut-off XML backup is still whole.
//
// Byline: Claude Code · Opus 5.5 · 2026-09-25
//
// Owner 2026-09-25 (repair workflow builder, option A): a truncated SMS
// backup is salvaged by keeping every record up to the last COMPLETE record
// element and closing the document, as a NEW derived artifact; the original
// is never written. See docs/pending-review/2026-09-25-repair-workflow-builder.md.
//
// Scanner is an io.Writer that follows the markup structure of whatever is
// written to it — start/end/self-closing tags, quoted attribute values,
// comments, CDATA, processing instructions and declarations — without keeping
// any content. Memory is constant: an MMS part whose base64 attribute runs to
// hundreds of megabytes costs nothing but the bytes passing through. It
// records the byte offset just past the last complete child of the root
// element, which is the cut point for a salvage.
//
// One unit, one job: it measures structure. It does not read the source,
// write the derived object, hash, or publish (AGENTS.md ATOMICITY).
package xmlsalvage

import (
	"bytes"
	"fmt"
)

const maxNameBytes = 256

type scanState int

const (
	stText scanState = iota
	stTagOpen
	stStartName
	stStartAttrs
	stEndTag
	stBang
	stComment
	stCDATA
	stPI
	stDecl
	stStopped
)

// Result is what the scanner learned. Offsets are absolute byte offsets into
// the stream written so far.
type Result struct {
	// Root is the document element's name ("smses", "calls", ...).
	Root string `json:"root"`
	// RootClosed is true when the document element's end tag was seen: the
	// document is complete, not truncated.
	RootClosed bool `json:"root_closed"`
	// Records counts complete children of the root element.
	Records uint64 `json:"records"`
	// LastCompleteEnd is the offset just past the '>' that closed the last
	// complete record. Zero when no record completed.
	LastCompleteEnd int64 `json:"last_complete_end"`
	// BytesSeen is the total number of bytes written to the scanner.
	BytesSeen int64 `json:"bytes_seen"`
	// Stopped names the malformation that ended structure tracking early, if
	// any. Records after it are not counted and not kept.
	Stopped string `json:"stopped,omitempty"`
	// StoppedAt is the offset of that malformation.
	StoppedAt int64 `json:"stopped_at,omitempty"`
}

// Truncated reports a document that was opened and never closed.
func (r Result) Truncated() bool { return r.Root != "" && !r.RootClosed }

// Scanner implements io.Writer. The zero value is ready to use.
type Scanner struct {
	offset int64
	state  scanState
	depth  int

	name       []byte
	quote      byte
	lastSignif byte // last non-space byte seen inside a start tag, outside quotes
	bang       []byte
	tail       [2]byte // the last two bytes seen inside a comment/CDATA/PI
	declDepth  int     // '[' nesting inside a <!DOCTYPE ...> declaration
	declQuote  byte

	result Result
}

// Result returns what the scanner has learned so far.
func (s *Scanner) Result() Result {
	out := s.result
	out.BytesSeen = s.offset
	return out
}

// Write consumes p. It never returns an error: a malformation stops structure
// tracking (recorded in Result.Stopped) but the bytes are still accepted, so
// the caller's copy of the stream is never cut short by the scanner.
func (s *Scanner) Write(p []byte) (int, error) {
	i := 0
	for i < len(p) {
		switch s.state {
		case stStopped:
			i = len(p)
		case stText:
			next := bytes.IndexByte(p[i:], '<')
			if next < 0 {
				i = len(p)
				continue
			}
			i += next + 1
			s.state = stTagOpen
		case stTagOpen:
			b := p[i]
			i++
			switch {
			case b == '/':
				s.name = s.name[:0]
				s.state = stEndTag
			case b == '?':
				s.tail = [2]byte{}
				s.state = stPI
			case b == '!':
				s.bang = s.bang[:0]
				s.state = stBang
			case isNameByte(b):
				s.name = append(s.name[:0], b)
				s.lastSignif = b
				s.state = stStartName
			default:
				s.stop(fmt.Sprintf("unexpected %q after '<'", b), i-1)
			}
		case stStartName:
			b := p[i]
			i++
			switch {
			case b == '>':
				s.finishStart(false, i)
			case b == '/':
				s.lastSignif = '/'
				s.state = stStartAttrs
			case isSpace(b):
				s.state = stStartAttrs
			case b == '<':
				s.stop("'<' inside a start tag", i-1)
			default:
				if len(s.name) < maxNameBytes {
					s.name = append(s.name, b)
				}
				s.lastSignif = b
			}
		case stStartAttrs:
			if s.quote != 0 {
				// XML forbids a raw '<' in an attribute value, so meeting one
				// means an unescaped quote desynchronised the scan (the same
				// rule SBV's decoder uses to resynchronise).
				stops := `"<`
				if s.quote == '\'' {
					stops = `'<`
				}
				next := bytes.IndexAny(p[i:], stops)
				if next < 0 {
					i = len(p)
					continue
				}
				if p[i+next] == '<' {
					s.stop("'<' inside an attribute value (an unescaped quote desynchronised attribute scanning)", i+next)
					continue
				}
				i += next + 1
				s.lastSignif = s.quote
				s.quote = 0
				continue
			}
			b := p[i]
			i++
			switch {
			case b == '"' || b == '\'':
				s.quote = b
			case b == '>':
				s.finishStart(s.lastSignif == '/', i)
			case b == '<':
				s.stop("'<' inside a start tag (an unescaped quote desynchronised attribute scanning)", i-1)
			case isSpace(b):
			default:
				s.lastSignif = b
			}
		case stEndTag:
			b := p[i]
			i++
			switch {
			case b == '>':
				s.finishEnd(i)
			case b == '<':
				s.stop("'<' inside an end tag", i-1)
			case isSpace(b):
			default:
				if len(s.name) < maxNameBytes {
					s.name = append(s.name, b)
				}
			}
		case stBang:
			b := p[i]
			i++
			s.bang = append(s.bang, b)
			switch {
			case bytes.Equal(s.bang, []byte("--")):
				s.tail = [2]byte{}
				s.state = stComment
			case bytes.Equal(s.bang, []byte("[CDATA[")):
				s.tail = [2]byte{}
				s.state = stCDATA
			case bytes.HasPrefix([]byte("--"), s.bang) || bytes.HasPrefix([]byte("[CDATA["), s.bang):
				// still ambiguous; keep reading
			default:
				s.declDepth, s.declQuote = 0, 0
				s.state = stDecl
				// Re-examine the bytes already consumed as declaration text.
				for _, c := range s.bang {
					if s.declByte(c) {
						break
					}
				}
			}
		case stComment:
			i = s.scanUntil(p, i, '-', '-', '>')
		case stCDATA:
			i = s.scanUntil(p, i, ']', ']', '>')
		case stPI:
			i = s.scanUntil(p, i, 0, '?', '>')
		case stDecl:
			b := p[i]
			i++
			s.declByte(b)
		}
	}
	s.offset += int64(len(p))
	return len(p), nil
}

// scanUntil advances through comment, CDATA or PI text until the terminator
// (a, b, end) — or (b, end) when a is 0 — and returns the next index.
func (s *Scanner) scanUntil(p []byte, i int, a, b, end byte) int {
	for i < len(p) {
		c := p[i]
		i++
		if c == end && s.tail[1] == b && (a == 0 || s.tail[0] == a) {
			s.tail = [2]byte{}
			s.state = stText
			return i
		}
		s.tail[0], s.tail[1] = s.tail[1], c
	}
	return i
}

// declByte consumes one byte of a <!DOCTYPE ...> style declaration and
// reports whether the declaration ended.
func (s *Scanner) declByte(c byte) bool {
	if s.declQuote != 0 {
		if c == s.declQuote {
			s.declQuote = 0
		}
		return false
	}
	switch c {
	case '"', '\'':
		s.declQuote = c
	case '[':
		s.declDepth++
	case ']':
		if s.declDepth > 0 {
			s.declDepth--
		}
	case '>':
		if s.declDepth == 0 {
			s.state = stText
			return true
		}
	}
	return false
}

// finishStart closes a start tag; end is the index just past its '>' within
// the current write, turned into an absolute offset here.
func (s *Scanner) finishStart(selfClosing bool, end int) {
	s.state = stText
	absolute := s.offset + int64(end)
	if s.result.RootClosed {
		// Markup after the document element is not part of the document.
		return
	}
	if s.depth == 0 {
		if s.result.Root != "" {
			s.stopAt("a second document element", absolute)
			return
		}
		s.result.Root = string(s.name)
		if selfClosing {
			s.result.RootClosed = true
			return
		}
		s.depth = 1
		return
	}
	if selfClosing {
		if s.depth == 1 {
			s.completeRecord(absolute)
		}
		return
	}
	s.depth++
}

func (s *Scanner) finishEnd(end int) {
	s.state = stText
	absolute := s.offset + int64(end)
	if s.result.RootClosed {
		return
	}
	if s.depth <= 0 {
		s.stopAt("an end tag with no open element", absolute)
		return
	}
	s.depth--
	switch s.depth {
	case 1:
		s.completeRecord(absolute)
	case 0:
		if string(s.name) != s.result.Root {
			s.stopAt(fmt.Sprintf("document element %q closed by </%s>", s.result.Root, s.name), absolute)
			return
		}
		s.result.RootClosed = true
	}
}

func (s *Scanner) completeRecord(end int64) {
	s.result.Records++
	s.result.LastCompleteEnd = end
}

func (s *Scanner) stop(reason string, at int) {
	s.stopAt(reason, s.offset+int64(at))
}

func (s *Scanner) stopAt(reason string, at int64) {
	s.result.Stopped = reason
	s.result.StoppedAt = at
	s.state = stStopped
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }

func isNameByte(b byte) bool {
	return b == '_' || b == ':' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b >= 0x80
}
