package acquisition

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/Cursedpotential/probata/engine/humancontext"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// HumanObjectSource reads native originals directly through the existing configured S3 clients.
// Inputs: existing scheme-to-client configuration. Outputs: pinned streaming source/range readers.
// Effects: GET requests only; no retention copy, hashing gate, repair or source writes.
type HumanObjectSource struct {Clients map[string]*s3.Client}

// Open resolves actual provider version, ETag and byte length while opening original bytes once.
// Inputs: registered source and optional known native pins. Outputs: owned reader and observed coordinates.
// Effects: read-only provider GET; choose after context registration rather than the retaining resolver.
func(s *HumanObjectSource)Open(ctx context.Context,source humancontext.Source)(io.ReadCloser,humancontext.Source,error){return s.open(ctx,source,0,0,false)}

// OpenRange opens one exact original half-open byte range for an attachment or validation readback.
// Inputs: verified manifest source/version/ETag pins and original offsets. Outputs: bounded raw reader.
// Effects: read-only provider range GET; bytes remain XML/base64 encoded until a separate decoder is requested.
func(s *HumanObjectSource)OpenRange(ctx context.Context,source humancontext.Source,start,end int64)(io.ReadCloser,error){
	if start<0||end<=start||source.ByteLength<=0||end>source.ByteLength{return nil,errors.New("invalid original range")}
	r,_,err:=s.open(ctx,source,start,end,true);return r,err
}
func(s *HumanObjectSource)open(ctx context.Context,source humancontext.Source,start,end int64,ranged bool)(io.ReadCloser,humancontext.Source,error){
	u,err:=url.Parse(source.SourceRef);if err!=nil||u.Host==""||u.User!=nil||u.RawQuery!=""||u.Fragment!=""{return nil,source,errors.New("invalid native object locator")}
	client:=s.Clients[u.Scheme];if client==nil{return nil,source,errors.New("native object store is not configured")}
	input:=&s3.GetObjectInput{Bucket:aws.String(u.Host),Key:aws.String(strings.TrimPrefix(u.Path,"/"))}
	if source.ProviderVersionID!=""{input.VersionId=aws.String(source.ProviderVersionID)}
	if source.ETag!=""{input.IfMatch=aws.String("\""+strings.Trim(source.ETag,"\"")+"\"")}
	if ranged{input.Range=aws.String(fmt.Sprintf("bytes=%d-%d",start,end-1))}
	output,err:=client.GetObject(ctx,input);if err!=nil{return nil,source,errors.New("native original read failed")}
	fail:=func()(io.ReadCloser,humancontext.Source,error){_ =output.Body.Close();return nil,source,errors.New("native original pin or range changed")}
	version,etag:=aws.ToString(output.VersionId),aws.ToString(output.ETag)
	if source.ProviderVersionID!=""&&version!=source.ProviderVersionID || source.ETag!=""&&!humancontext.SameETag(etag,source.ETag){return fail()}
	if version==""&&etag==""{return fail()}
	if ranged{if aws.ToInt64(output.ContentLength)!=end-start || aws.ToString(output.ContentRange)!=fmt.Sprintf("bytes %d-%d/%d",start,end-1,source.ByteLength){return fail()}}else{actual:=aws.ToInt64(output.ContentLength);if actual<0||(source.ByteLength>0&&source.ByteLength!=actual){return fail()};source.ByteLength=actual}
	source.ProviderVersionID=version;source.ETag=etag
	return output.Body,source,nil
}
