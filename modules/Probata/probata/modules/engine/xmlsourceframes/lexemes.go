package xmlsourceframes

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
)

// Lexeme describes XML syntax with original half-open byte ranges, never message semantics.
// Attribute values below the cap carry exact escaped bytes as base64; larger values carry only ranges.
// Element IDs, parent IDs and source ordinals join lexical rows in DuckDB without relying on timestamps.
// Source/version identity belongs to the manifest containing these rows, not an invented retained object.
type Lexeme struct {
	Kind           string `json:"kind"`
	ElementID      int64  `json:"element_id"`
	ParentID       int64  `json:"parent_id"`
	RecordOrdinal  int64  `json:"record_ordinal"`
	Name           string `json:"name"`
	Start          int64  `json:"start"`
	End            int64  `json:"end"`
	ValueStart     int64  `json:"value_start,omitempty"`
	ValueEnd       int64  `json:"value_end,omitempty"`
	ValueBytes     int64  `json:"value_bytes,omitempty"`
	RawValueBase64 string `json:"raw_value_b64,omitempty"`
	Quote          string `json:"quote,omitempty"`
	RangeOnly      bool   `json:"range_only,omitempty"`
	SelfClosing    bool   `json:"self_closing,omitempty"`
}

type lexicalElement struct{ id, parent, ordinal int64 }
type lexicalState struct {
	emit                  func(Lexeme) error
	limit                 int
	nextID                int64
	stack                 []lexicalElement
	current               lexicalElement
	attr                  []byte
	attrStart, valueStart int64
	value                 []byte
	quote                 byte
	rangeOnly             bool
	textStart             int64
	textNonspace          bool
}

// finishLexicalText preserves non-whitespace character content as an original range.
// Inputs: original end offset. Outputs: lexical reference only; effects: streaming sink call.
// DuckDB marks unsupported native character content pending rather than silently discarding it.
func (s *scanner) finishLexicalText(end int64) error {
	if s.lex == nil {
		return nil
	}
	l := s.lex
	defer func() { l.textNonspace = false }()
	if !l.textNonspace || len(l.stack) == 0 {
		return nil
	}
	parent := l.stack[len(l.stack)-1]
	return l.emit(Lexeme{Kind: "text", Name: "#text", ParentID: parent.id, RecordOrdinal: parent.ordinal, Start: l.textStart, End: end, RangeOnly: true})
}

// opaqueLexicalContent preserves CDATA without materializing or interpreting its payload.
// Inputs: original declaration range. Outputs: lexical reference; effects: streaming sink call.
func (s *scanner) opaqueLexicalContent() error {
	if s.lex == nil || len(s.lex.stack) == 0 {
		return nil
	}
	parent := s.lex.stack[len(s.lex.stack)-1]
	return s.lex.emit(Lexeme{Kind: "cdata", Name: "#cdata", ParentID: parent.id, RecordOrdinal: parent.ordinal, Start: s.tagStart, End: s.offset, RangeOnly: true})
}

// ScanLexemes emits bounded generic attribute syntax alongside disjoint element framing.
// Inputs: owned cancellable original reader, frame/value caps and two streaming sinks.
// Outputs: structural receipt and original lexical references; effects: reads/closes reader and calls sinks.
// Pick for DuckDB input preparation, never as a Go SMS parser. XML entity/encoding validation remains downstream.
func ScanLexemes(ctx context.Context, reader io.ReadCloser, maxFrameBytes int64, maxValueBytes int, spans func(Span) error, lexemes func(Lexeme) error) (Receipt, error) {
	if maxValueBytes < 1 || maxValueBytes > 64*1024 || lexemes == nil {
		return Receipt{}, errors.New("invalid lexical value bound")
	}
	return scan(ctx, reader, maxFrameBytes, spans, &lexicalState{emit: lexemes, limit: maxValueBytes})
}

func (s *scanner) beginLexicalTag() {
	if s.lex == nil {
		return
	}
	l := s.lex
	if s.closing {
		if len(l.stack) > 0 {
			l.current = l.stack[len(l.stack)-1]
		}
		return
	}
	l.nextID++
	l.current = lexicalElement{id: l.nextID}
	if len(l.stack) > 0 {
		l.current.parent = l.stack[len(l.stack)-1].id
	}
	if len(s.stack) > 0 {
		l.current.ordinal = s.ordinal + 1
	}
}

func (s *scanner) finishLexicalTag() error {
	if s.lex == nil {
		return nil
	}
	l := s.lex
	kind := "element_start"
	if s.closing {
		kind = "element_end"
	}
	if err := l.emit(Lexeme{Kind: kind, ElementID: l.current.id, ParentID: l.current.parent, RecordOrdinal: l.current.ordinal, Name: string(s.name), Start: s.tagStart, End: s.offset, SelfClosing: !s.closing && s.last == '/'}); err != nil {
		return err
	}
	if s.closing {
		if len(l.stack) > 0 {
			l.stack = l.stack[:len(l.stack)-1]
		}
	} else if s.last != '/' {
		l.stack = append(l.stack, l.current)
	}
	return nil
}

func (s *scanner) consumeLexicalAttribute(b byte) error {
	l := s.lex
	switch s.mode {
	case "attrs":
		if whitespace(b) {
			return nil
		}
		if b == '>' {
			return s.finishTag()
		}
		if b == '/' {
			s.last = b
			return nil
		}
		if s.closing || s.last == '/' {
			s.fail("unexpected_attribute")
			return nil
		}
		l.attr = []byte{b}
		l.attrStart = s.offset - 1
		s.mode = "lex_attr_name"
	case "lex_attr_name":
		if b == '=' {
			s.mode = "lex_attr_quote"
		} else if whitespace(b) {
			s.mode = "lex_attr_equals"
		} else if len(l.attr) >= maxNameBytes || b == '<' || b == '>' || b == '/' {
			s.fail("invalid_attribute_name")
		} else {
			l.attr = append(l.attr, b)
		}
	case "lex_attr_equals":
		if b == '=' {
			s.mode = "lex_attr_quote"
		} else if !whitespace(b) {
			s.fail("missing_attribute_equals")
		}
	case "lex_attr_quote":
		if b == '\'' || b == '"' {
			l.quote = b
			l.valueStart = s.offset
			l.value = nil
			l.rangeOnly = false
			s.mode = "lex_attr_value"
		} else if !whitespace(b) {
			s.fail("missing_attribute_quote")
		}
	case "lex_attr_value":
		if b == l.quote {
			row := Lexeme{Kind: "attribute", ElementID: l.current.id, ParentID: l.current.parent, RecordOrdinal: l.current.ordinal, Name: string(l.attr), Start: l.attrStart, End: s.offset, ValueStart: l.valueStart, ValueEnd: s.offset - 1, ValueBytes: s.offset - 1 - l.valueStart, Quote: string(l.quote), RangeOnly: l.rangeOnly}
			if !l.rangeOnly {
				row.RawValueBase64 = base64.StdEncoding.EncodeToString(l.value)
			}
			if err := l.emit(row); err != nil {
				return err
			}
			l.value = nil
			s.mode = "lex_after_value"
			s.last = 0
		} else if !l.rangeOnly {
			if len(l.value) >= l.limit {
				l.value = nil
				l.rangeOnly = true
			} else {
				l.value = append(l.value, b)
			}
		}
	case "lex_after_value":
		if whitespace(b) {
			s.mode = "attrs"
		} else if b == '>' {
			return s.finishTag()
		} else if b == '/' {
			s.last = b
			s.mode = "attrs"
		} else {
			s.fail("missing_attribute_separator")
		}
	}
	return nil
}
