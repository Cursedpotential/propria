// Package xmlsourceframes measures original XML element spans without interpreting source fields.
// Inputs are original bytes; outputs contain offsets only. It never repairs, decodes, hashes or writes originals.
package xmlsourceframes

import (
	"context"
	"errors"
	"io"
	"sync"
)

const bufferBytes = 64 * 1024
const maxNameBytes = 256
const maxDepth = 256

// Span identifies one disjoint original range using half-open byte offsets.
// Element names remain lexical qualified names; oversized elements retain their entire range.
// This is structural framing, not XML validation or semantic message extraction.
type Span struct {
	Start   int64  `json:"start"`
	End     int64  `json:"end"`
	Kind    string `json:"kind"`
	Name    string `json:"name,omitempty"`
	Ordinal int64  `json:"ordinal,omitempty"`
	Status  string `json:"status"`
}

// Receipt records complete stream accounting without source content.
// A malformed remainder is preserved as an unparsed span; callers must not claim complete extraction.
type Receipt struct {
	Bytes     int64  `json:"bytes"`
	Elements  int64  `json:"elements"`
	Oversized int64  `json:"oversized"`
	Complete  bool   `json:"complete"`
	Reason    string `json:"reason,omitempty"`
}

type scanner struct {
	emit                               func(Span) error
	limit                              int64
	offset, segment, tagStart, ordinal int64
	mode                               string
	name                               []byte
	stack                              []string
	quote, last                        byte
	tail                               string
	declarationDepth                   int
	closing, rootSeen, rootClosed      bool
	recordName                         string
	receipt                            Receipt
	lex                                *lexicalState
}

// Scan frames direct children of the document element with fixed-size input buffers.
// Inputs: cancellable owned reader, ordinary frame limit and a streaming offset sink.
// Outputs: disjoint spans covering every consumed byte and a reference-only receipt.
// Effects: reads and closes reader, invokes sink; cancellation closes blocked reads.
// Choose for extraction preparation; do not substitute display-truncated preview rows.
func Scan(ctx context.Context, reader io.ReadCloser, maxFrameBytes int64, emit func(Span) error) (Receipt, error) {
	return scan(ctx, reader, maxFrameBytes, emit, nil)
}

func scan(ctx context.Context, reader io.ReadCloser, maxFrameBytes int64, emit func(Span) error, lex *lexicalState) (Receipt, error) {
	if reader == nil || emit == nil || maxFrameBytes <= 0 {
		return Receipt{}, errors.New("invalid framing input")
	}
	done := make(chan struct{})
	joined := make(chan struct{})
	var closeOnce sync.Once
	closeReader := func() { closeOnce.Do(func() { _ = reader.Close() }) }
	go func() {
		defer close(joined)
		select {
		case <-ctx.Done():
			closeReader()
		case <-done:
		}
	}()
	defer func() { close(done); closeReader(); <-joined }()
	s := scanner{emit: emit, limit: maxFrameBytes, mode: "text", lex: lex}
	buffer := make([]byte, bufferBytes)
	for {
		if err := ctx.Err(); err != nil {
			return s.receipt, err
		}
		n, err := reader.Read(buffer)
		for i, b := range buffer[:n] {
			if i%4096 == 0 {
				if e := ctx.Err(); e != nil {
					return s.receipt, e
				}
			}
			s.offset++
			if e := s.consume(b); e != nil {
				return s.receipt, e
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				return s.receipt, ctx.Err()
			}
			return s.receipt, err
		}
		if n == 0 {
			return s.receipt, io.ErrNoProgress
		}
	}
	s.receipt.Bytes = s.offset
	if err:=s.finishLexicalText(s.offset);err!=nil{return s.receipt,err}
	if s.receipt.Reason == "" && (!s.rootClosed || s.mode != "text") {
		s.fail("incomplete_structure")
	}
	if s.segment < s.offset {
		kind, status := "envelope", "complete"
		if s.receipt.Reason != "" {
			kind, status = "unparsed", "malformed"
		}
		if err := s.emit(Span{Start: s.segment, End: s.offset, Kind: kind, Status: status}); err != nil {
			return s.receipt, err
		}
	}
	s.receipt.Complete = s.receipt.Reason == ""
	return s.receipt, nil
}

func (s *scanner) fail(reason string) {
	if s.receipt.Reason == "" {
		s.receipt.Reason = reason
	}
	s.mode = "failed"
}
func whitespace(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }
func (s *scanner) addName(b byte) {
	if len(s.name) >= maxNameBytes {
		s.fail("name_limit")
		return
	}
	s.name = append(s.name, b)
}
func (s *scanner) envelope(end int64) error {
	if end > s.segment {
		if err := s.emit(Span{Start: s.segment, End: end, Kind: "envelope", Status: "complete"}); err != nil {
			return err
		}
	}
	s.segment = end
	return nil
}
func (s *scanner) finishTag() error {
	name := string(s.name)
	if name == "" {
		s.fail("empty_name")
		return nil
	}
	if s.closing {
		if len(s.stack) == 0 || s.stack[len(s.stack)-1] != name {
			s.fail("mismatched_close")
			return nil
		}
		if err := s.finishLexicalTag(); err != nil { return err }
		s.stack = s.stack[:len(s.stack)-1]
		if len(s.stack) == 1 {
			if err := s.element(); err != nil {
				return err
			}
		}
		if len(s.stack) == 0 {
			s.rootClosed = true
		}
	} else {
		if s.rootClosed || (len(s.stack) == 0 && s.rootSeen) {
			s.fail("multiple_roots")
			return nil
		}
		if err := s.finishLexicalTag(); err != nil { return err }
		if len(s.stack) == 0 {
			s.rootSeen = true
		}
		if len(s.stack) == 1 {
			if err := s.envelope(s.tagStart); err != nil {
				return err
			}
			s.recordName = name
		}
		if s.last != '/' {
			if len(s.stack) >= maxDepth {
				s.fail("depth_limit")
				return nil
			}
			s.stack = append(s.stack, name)
		} else if len(s.stack) == 1 {
			if err := s.element(); err != nil {
				return err
			}
		} else if len(s.stack) == 0 {
			s.rootClosed = true
		}
	}
	s.mode = "text"
	return nil
}
func (s *scanner) element() error {
	s.ordinal++
	s.receipt.Elements++
	status := "complete"
	if s.offset-s.segment > s.limit {
		status = "oversized"
		s.receipt.Oversized++
	}
	if err := s.emit(Span{Start: s.segment, End: s.offset, Kind: "element", Name: s.recordName, Ordinal: s.ordinal, Status: status}); err != nil {
		return err
	}
	s.segment = s.offset
	return nil
}
func (s *scanner) consume(b byte) error {
	switch s.mode {
	case "failed":
		return nil
	case "text":
		if b == '<' {
			if err:=s.finishLexicalText(s.offset-1);err!=nil{return err}
			s.tagStart = s.offset - 1
			s.mode = "open"
			s.name = nil
			s.closing = false
			s.last = 0
			s.quote = 0
		}else if s.lex!=nil&&!whitespace(b)&&!s.lex.textNonspace{s.lex.textStart=s.offset-1;s.lex.textNonspace=true
		}
	case "open":
		switch b {
		case '/':
			s.closing = true
			s.mode = "name"
		case '?':
			s.mode = "pi"
			s.tail = ""
		case '!':
			s.mode = "bang"
			s.tail = ""
		default:
			s.mode = "name"
			s.addName(b)
		}
	case "name":
		if whitespace(b) {
			s.beginLexicalTag()
			s.mode = "attrs"
		} else if b == '>' {
			s.beginLexicalTag()
			return s.finishTag()
		} else if b == '/' {
			s.beginLexicalTag()
			s.last = b
			s.mode = "attrs"
		} else {
			s.addName(b)
		}
	case "attrs":
		if s.lex != nil { return s.consumeLexicalAttribute(b) }
		if s.quote != 0 {
			if b == s.quote {
				s.quote = 0
			}
		} else if b == '\'' || b == '"' {
			s.quote = b
		} else if b == '>' {
			return s.finishTag()
		} else if !whitespace(b) {
			s.last = b
		}
	case "lex_attr_name", "lex_attr_equals", "lex_attr_quote", "lex_attr_value", "lex_after_value":
		return s.consumeLexicalAttribute(b)
	case "bang":
		s.tail += string(b)
		if s.tail == "--" {
			s.mode = "comment"
			s.tail = ""
		} else if s.tail == "[CDATA[" {
			s.mode = "cdata"
			s.tail = ""
		} else if len(s.tail) >= 7 {
			s.mode = "decl"
			s.declarationDepth = 0
			s.quote = 0
		}
	case "comment", "cdata", "pi":
		s.tail += string(b)
		if len(s.tail) > 3 {
			s.tail = s.tail[len(s.tail)-3:]
		}
		if s.mode == "comment" && s.tail == "-->" || s.mode == "cdata" && s.tail == "]]>" || s.mode == "pi" && len(s.tail) >= 2 && s.tail[len(s.tail)-2:] == "?>" {
			if s.mode=="cdata"{if err:=s.opaqueLexicalContent();err!=nil{return err}}
			s.mode = "text"
		}
	case "decl":
		if s.quote != 0 {
			if b == s.quote {
				s.quote = 0
			}
		} else if b == '\'' || b == '"' {
			s.quote = b
		} else if b == '[' {
			s.declarationDepth++
		} else if b == ']' {
			s.declarationDepth--
		} else if b == '>' && s.declarationDepth == 0 {
			s.mode = "text"
		}
	}
	return nil
}
