package sourcepreview

import (
	"bufio"
	"context"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

// State is a bounded lexical continuation held only in the private HTTP cache.
// Inputs/outputs: absolute byte offset and bounded partial record fields. Effects: none.
// Choose it over encoding/xml.Token for raw backups whose single attribute can exceed memory budgets.
type State struct {
	Offset          int64             `json:"offset"`
	SourceLength    int64             `json:"source_length,omitempty"`
	Mode            string            `json:"mode,omitempty"`
	Tag             string            `json:"tag,omitempty"`
	Attr            string            `json:"attr,omitempty"`
	Value           []byte            `json:"value,omitempty"`
	ValueBytes      int64             `json:"value_bytes,omitempty"`
	Quote           byte              `json:"quote,omitempty"`
	Attrs           map[string]string `json:"attrs,omitempty"`
	AttrTruncated   bool              `json:"attr_truncated,omitempty"`
	DataBytes       int64             `json:"data_bytes,omitempty"`
	Stack           []string          `json:"stack,omitempty"`
	RootSeen        bool              `json:"root_seen,omitempty"`
	Done            bool              `json:"done,omitempty"`
	RecordStart     int64             `json:"record_start,omitempty"`
	TagStart        int64             `json:"tag_start,omitempty"`
	RecordTags      int               `json:"record_tags,omitempty"`
	RecordOversized bool              `json:"record_oversized,omitempty"`
	Current         *Message          `json:"current,omitempty"`
	Special         string            `json:"special,omitempty"`
}

// Scan reads one bounded raw XML page and returns its lexical continuation.
// Inputs: cancellation context, a pinned range reader, source length, private continuation and row limit.
// Outputs: bounded message views, scanned bytes and a resume state; effects: reads at most MaxScanBytes.
// Choose before ingestion to inspect SMS/MMS attributes without decoding attachment payloads.
func Scan(ctx context.Context, reader io.Reader, source Source, state State, limit int) (Completion, error) {
	if limit <= 0 || limit > MaxRows || state.Offset < 0 || state.Offset > source.ByteLength {
		return Completion{}, ErrInvalid
	}
	if state.Offset > 0 && state.SourceLength != source.ByteLength {
		return Completion{}, ErrChanged
	}
	state.SourceLength = source.ByteLength
	page := Page{Status: "ready", Source: source, Messages: []Message{}, PreviewOnly: true}
	start := state.Offset
	buffer := bufio.NewReaderSize(io.LimitReader(reader, MaxScanBytes), 8192)
	for state.Offset-start < MaxScanBytes && !state.Done && len(page.Messages) < limit {
		if state.Offset%4096 == 0 {
			if err := ctx.Err(); err != nil {
				return Completion{}, err
			}
		}
		b, err := buffer.ReadByte()
		if err == io.EOF {
			if state.Offset < source.ByteLength && state.Offset-start < MaxScanBytes {
				return Completion{}, ErrUnavailable
			}
			break
		}
		if err != nil {
			return Completion{}, ErrUnavailable
		}
		state.Offset++
		if !state.RecordOversized && (state.Current != nil && state.Offset-state.RecordStart > MaxRecordBytes || len(state.Stack) == 1 && (state.Tag == "sms" || state.Tag == "mms") && state.Offset-state.TagStart > MaxRecordBytes) {
			// The record inspection cap switches to lexical skipping, rather than
			// hiding every following message behind one enormous attachment.
			state.RecordOversized = true
			state.Value = nil
			if state.Current != nil {
				state.Current.BodyTruncated = true
				state.Current.MetadataTruncated = true
			}
		}
		message, err := state.consume(b)
		if err != nil {
			return Completion{}, ErrInvalid
		}
		if message != nil {
			page.Messages = append(page.Messages, *message)
		}
	}
	if err := ctx.Err(); err != nil {
		return Completion{}, err
	}
	page.ScannedBytes = state.Offset - start
	if state.Offset == source.ByteLength && !state.Done {
		return Completion{}, ErrInvalid
	}
	if !state.Done {
		if len(page.Messages) == limit {
			page.TruncationReason = "row_limit"
		} else {
			page.TruncationReason = "scan_byte_limit"
		}
	}
	return Completion{Page: page, State: state}, nil
}

func space(b byte) bool { return b == ' ' || b == '\t' || b == '\r' || b == '\n' }
func nameByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '-' || b == ':'
}
func boundedName(value string, b byte) (string, error) {
	if !nameByte(b) || len(value) >= 64 {
		return "", ErrInvalid
	}
	return value + string(b), nil
}

func (s *State) consume(b byte) (*Message, error) {
	switch s.Mode {
	case "", "text":
		if b == '<' {
			s.TagStart = s.Offset - 1
			s.Mode = "open"
			s.Tag = ""
			s.Attrs = nil
			s.DataBytes = 0
			s.AttrTruncated = false
		} else if !space(b) && !(s.Offset <= 3 && (b == 0xef || b == 0xbb || b == 0xbf)) {
			return nil, ErrInvalid
		}
	case "open":
		switch b {
		case '/':
			s.Mode = "close"
		case '?':
			s.Mode = "pi"
			s.Special = ""
		case '!':
			s.Mode = "bang"
			s.Special = ""
		default:
			if !nameByte(b) {
				return nil, ErrInvalid
			}
			s.Tag = string(b)
			s.Mode = "tag"
		}
	case "pi":
		if s.Special == "?" && b == '>' {
			s.Mode = "text"
		}
		s.Special = string(b)
	case "bang":
		s.Special += string(b)
		if !strings.HasPrefix("--", s.Special) {
			return nil, ErrInvalid
		}
		if s.Special == "--" {
			s.Mode = "comment"
			s.Special = ""
		}
	case "comment":
		s.Special += string(b)
		if len(s.Special) > 3 {
			s.Special = s.Special[len(s.Special)-3:]
		}
		if s.Special == "-->" {
			s.Mode = "text"
			s.Special = ""
		}
	case "tag":
		if space(b) {
			s.Mode = "between"
		} else if b == '>' {
			return s.start(false)
		} else if b == '/' {
			s.Mode = "selfclose"
		} else {
			var err error
			s.Tag, err = boundedName(s.Tag, b)
			return nil, err
		}
	case "between":
		if space(b) {
			break
		}
		if b == '>' {
			return s.start(false)
		}
		if b == '/' {
			s.Mode = "selfclose"
			break
		}
		if !nameByte(b) {
			return nil, ErrInvalid
		}
		s.Attr = string(b)
		s.Mode = "attr"
	case "attr":
		if b == '=' {
			s.Mode = "quote"
		} else if space(b) {
			s.Mode = "equals"
		} else {
			var err error
			s.Attr, err = boundedName(s.Attr, b)
			return nil, err
		}
	case "equals":
		if space(b) {
			break
		}
		if b != '=' {
			return nil, ErrInvalid
		}
		s.Mode = "quote"
	case "quote":
		if space(b) {
			break
		}
		if b != '\'' && b != '"' {
			return nil, ErrInvalid
		}
		s.Quote = b
		s.Value = nil
		s.ValueBytes = 0
		s.Mode = "value"
	case "value":
		if b == s.Quote {
			if wanted(s.Attr) && !s.RecordOversized {
				if s.Attrs == nil {
					s.Attrs = map[string]string{}
				}
				if len(s.Attrs) >= 24 {
					return nil, ErrInvalid
				}
				value, err := decodeAttribute(string(s.Value), s.Quote, s.ValueBytes > MaxAttributeBytes)
				if err != nil {
					return nil, err
				}
				s.Attrs[s.Attr] = value
			}
			if s.Attr == "data" {
				s.DataBytes = s.ValueBytes
			}
			if s.ValueBytes > MaxAttributeBytes && wanted(s.Attr) {
				s.AttrTruncated = true
			}
			s.Mode = "between"
			s.Value = nil
			s.Attr = ""
			s.ValueBytes = 0
		} else {
			if b == '<' {
				return nil, ErrInvalid
			}
			s.ValueBytes++
			if wanted(s.Attr) && !s.RecordOversized && len(s.Value) < MaxAttributeBytes {
				s.Value = append(s.Value, b)
			}
		}
	case "selfclose":
		if b != '>' {
			return nil, ErrInvalid
		}
		return s.start(true)
	case "close":
		if b == '>' {
			return s.close()
		}
		if space(b) {
			s.Mode = "closewait"
		} else {
			var err error
			s.Tag, err = boundedName(s.Tag, b)
			return nil, err
		}
	case "closewait":
		if b == '>' {
			return s.close()
		}
		if !space(b) {
			return nil, ErrInvalid
		}
	default:
		return nil, ErrInvalid
	}
	return nil, nil
}

func wanted(attr string) bool {
	switch attr {
	case "date", "type", "msg_box", "address", "body", "text", "ct", "cl", "name", "filename":
		return true
	}
	return false
}
func decodeAttribute(raw string, quote byte, truncated bool) (string, error) {
	if truncated {
		for !utf8.ValidString(raw) && len(raw) > 0 {
			raw = raw[:len(raw)-1]
		}
		if i := strings.LastIndex(raw, "&"); i >= 0 && !strings.Contains(raw[i:], ";") {
			raw = raw[:i]
		}
	}
	decoder := xml.NewDecoder(strings.NewReader("<x a=" + string(quote) + raw + string(quote) + "/>"))
	token, err := decoder.Token()
	if err != nil {
		return "", ErrInvalid
	}
	return token.(xml.StartElement).Attr[0].Value, nil
}
func clip(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) && len(value) > 0 {
		value = value[:len(value)-1]
	}
	return value
}

func (s *State) start(selfClosing bool) (*Message, error) {
	s.Mode = "text"
	if !s.RootSeen {
		if s.Tag != "smses" || len(s.Stack) != 0 || selfClosing {
			return nil, ErrInvalid
		}
		s.RootSeen = true
	} else if len(s.Stack) == 0 {
		return nil, ErrInvalid
	}
	if len(s.Stack) == 1 {
		if s.Tag != "sms" && s.Tag != "mms" {
			return nil, ErrInvalid
		}
		if s.Current != nil {
			return nil, ErrInvalid
		}
		s.RecordStart = s.TagStart
		s.RecordTags = 0
		m := &Message{ID: strconv.FormatInt(s.TagStart, 10), Kind: s.Tag, Direction: "unknown", Participants: []string{}, Attachments: []Attachment{}}
		if date, err := strconv.ParseInt(s.Attrs["date"], 10, 64); err == nil && date >= 0 && date <= 8640000000000000 {
			m.DateMS = &date
		}
		typ := s.Attrs["type"]
		if s.Tag == "mms" {
			typ = s.Attrs["msg_box"]
		}
		if typ == "1" {
			m.Direction = "incoming"
		}
		if typ == "2" {
			m.Direction = "outgoing"
		}
		m.Body = clip(s.Attrs["body"], MaxBodyBytes)
		m.BodyTruncated = s.AttrTruncated || len(s.Attrs["body"]) > MaxBodyBytes
		if s.RecordOversized {
			m.BodyTruncated = true
			m.MetadataTruncated = true
		}
		if address := s.Attrs["address"]; address != "" && address != "null" {
			m.Participants = append(m.Participants, clip(address, 128))
			if len(address) > 128 {
				m.MetadataTruncated = true
			}
		}
		s.Current = m
	}
	if s.Current != nil {
		s.RecordTags++
		if s.RecordTags > MaxTags {
			return nil, ErrInvalid
		}
		s.Current.MetadataTruncated = s.Current.MetadataTruncated || s.AttrTruncated
		if s.Tag == "addr" && !s.RecordOversized {
			if value := s.Attrs["address"]; value != "" && value != "insert-address-token" {
				if len(s.Current.Participants) < MaxParticipants {
					s.Current.Participants = append(s.Current.Participants, clip(value, 128))
					if len(value) > 128 {
						s.Current.MetadataTruncated = true
					}
				} else {
					s.Current.MetadataTruncated = true
				}
			}
		}
		if s.Tag == "part" && !s.RecordOversized {
			ct := s.Attrs["ct"]
			if ct == "text/plain" {
				text := s.Attrs["text"]
				left := MaxBodyBytes - len(s.Current.Body)
				if len(text) > left || s.AttrTruncated {
					s.Current.BodyTruncated = true
				}
				s.Current.Body += clip(text, left)
			} else {
				if len(s.Current.Attachments) < MaxAttachments {
					name := s.Attrs["cl"]
					if name == "" {
						name = s.Attrs["name"]
					}
					if name == "" {
						name = s.Attrs["filename"]
					}
					s.Current.Attachments = append(s.Current.Attachments, Attachment{ContentType: clip(ct, 128), Name: clip(name, 128), EncodedBytes: s.DataBytes})
					if len(ct) > 128 || len(name) > 128 {
						s.Current.MetadataTruncated = true
					}
				} else {
					s.Current.MetadataTruncated = true
				}
			}
		}
	}
	if len(s.Stack) >= MaxDepth {
		return nil, ErrInvalid
	}
	s.Stack = append(s.Stack, s.Tag)
	if selfClosing {
		return s.close()
	}
	return nil, nil
}
func (s *State) close() (*Message, error) {
	s.Mode = "text"
	if len(s.Stack) == 0 || s.Stack[len(s.Stack)-1] != s.Tag {
		return nil, ErrInvalid
	}
	s.Stack = s.Stack[:len(s.Stack)-1]
	if len(s.Stack) == 0 {
		s.Done = true
	}
	if (s.Tag == "sms" || s.Tag == "mms") && len(s.Stack) == 1 {
		m := s.Current
		s.Current = nil
		s.Attrs = nil
		s.RecordTags = 0
		s.RecordOversized = false
		return m, nil
	}
	return nil, nil
}
