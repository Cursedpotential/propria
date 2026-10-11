package xmlsourceframes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestLexicalValuesPreserveExactOriginalBytesAndHierarchy(t *testing.T){
	input:=`<root xmlns:x="urn:native"><x:item body="é &amp; &#10;" empty=''><child value='native'/></x:item><following/></root>`
	var rows []Lexeme
	r,err:=ScanLexemes(context.Background(),io.NopCloser(strings.NewReader(input)),1024,1024,func(Span)error{return nil},func(l Lexeme)error{rows=append(rows,l);return nil})
	if err!=nil || !r.Complete{t.Fatalf("%+v %v",r,err)}
	for _,row:=range rows{if row.Kind=="attribute"{value,err:=base64.StdEncoding.DecodeString(row.RawValueBase64);if err!=nil || string(value)!=input[row.ValueStart:row.ValueEnd]{t.Fatal(row,err)};if row.ValueEnd-row.ValueStart!=row.ValueBytes{t.Fatal(row)}}}
	if len(rows)!=10 {t.Fatalf("rows=%d %+v",len(rows),rows)}
	var child,item Lexeme;for _,row:=range rows{if row.Kind=="element_start"&&row.Name=="child"{child=row};if row.Kind=="element_start"&&row.Name=="x:item"{item=row}}
	if child.ParentID!=item.ElementID || child.RecordOrdinal!=1 || item.RecordOrdinal!=1{t.Fatal(item,child)}
}

func TestLexicalGiantMMSAttachmentAndFollowingSMS(t *testing.T){
	prefix:=`<smses><sms address="123" type="1" date="1700000000000" body="é &amp; hello"/><mms date="1700000001000" msg_box="1"><parts><part seq="0" ct="image/png" data="`
	suffix:=`"/><part seq="1" ct="text/plain" text="caption &amp; text"/></parts><addrs><addr address="123" type="137"/><addr address="456" type="151"/></addrs></mms><sms address="456" type="2" date="1700000002000" body="following"/></smses>`
	const payload=int64(65*1024*1024)
	reader:=io.MultiReader(strings.NewReader(prefix),&repeatedByte{remaining:payload},strings.NewReader(suffix))
	var rows []Lexeme
	r,err:=ScanLexemes(context.Background(),io.NopCloser(reader),1024,1024,func(Span)error{return nil},func(l Lexeme)error{rows=append(rows,l);return nil})
	if err!=nil || !r.Complete || r.Elements!=3 || r.Oversized!=1{t.Fatalf("%+v %v",r,err)}
	var data Lexeme;following:=false
	for _,row:=range rows{if row.Kind=="attribute"&&row.Name=="data"{data=row};if row.Kind=="element_start"&&row.Name=="sms"&&row.RecordOrdinal==3{following=true};if len(row.RawValueBase64)>1400{t.Fatal("unbounded value")}}
	if !data.RangeOnly || data.RawValueBase64!="" || data.ValueBytes!=payload || data.ValueEnd-data.ValueStart!=payload || !following{t.Fatal(data,following)}
	// An explicitly requested private fixture path allows the remote DuckDB test
	// to consume these synthetic lexical rows; normal tests write nothing.
	if path:=os.Getenv("SYNTHETIC_LEXEME_FIXTURE");path!=""{file,err:=os.Create(path);if err!=nil{t.Fatal(err)};defer file.Close();encoder:=json.NewEncoder(file);for _,row:=range rows{if err:=encoder.Encode(row);err!=nil{t.Fatal(err)}}}
	t.Logf("bytes=%d lexemes=%d attachment_start=%d attachment_end=%d following=%v",r.Bytes,len(rows),data.ValueStart,data.ValueEnd,following)
}

func TestLexicalInvalidAttributeSyntaxRemainsUnparsed(t *testing.T){
	for _,input:=range []string{`<root><item a='x'b='y'/></root>`,`<root><item a=x/></root>`,`<root><item a 'x'/></root>`}{r,err:=ScanLexemes(context.Background(),io.NopCloser(strings.NewReader(input)),1024,1024,func(Span)error{return nil},func(Lexeme)error{return nil});if err!=nil || r.Complete{t.Fatalf("%q %+v %v",input,r,err)}}
}
