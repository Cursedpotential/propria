package weaviate

import (
 "context"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "testing"
 "github.com/Cursedpotential/probata/engine/humancontext"
)

// TestHumanTargetUsesBatchUpsertAndIndependentGet verifies the existing idempotent transport and per-object rejection.
// Inputs: synthetic server/object only. Outputs: matching GET or error; effects: loopback HTTP, no real search writes.
func TestHumanTargetUsesBatchUpsertAndIndependentGet(t *testing.T){
 const id="018f0000-0000-7000-8000-000000000001"
 fail:=false;writes,reads:=0,0
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){switch r.URL.Path{case "/v1/batch/objects":
 if r.Method!=http.MethodPost{t.Error("not batch POST")};writes++;var body struct{Objects []map[string]any `json:"objects"`};if err:=json.NewDecoder(r.Body).Decode(&body);err!=nil{t.Error(err)};if len(body.Objects)!=1||body.Objects[0]["id"]!=id{t.Errorf("batch %+v",body)}
 status:="SUCCESS";if fail{status="FAILED"};_ =json.NewEncoder(w).Encode([]any{map[string]any{"id":id,"result":map[string]any{"status":status}}})
 case "/v1/objects/MsgEvents20260918/"+id:reads++;_ =json.NewEncoder(w).Encode(map[string]any{"id":id,"class":"MsgEvents20260918","properties":map[string]any{"body":"synthetic"},"vectors":map[string]any{"text_nim":[]float32{.25,.5}}})
 default:http.NotFound(w,r)}}));defer server.Close()
 target:=&HumanSourceTarget{Store:Store{BaseURL:server.URL,Collection:"MsgEvents20260918",VectorName:"text_nim"}}
 object:=humancontext.SearchObject{ID:id,Properties:map[string]any{"body":"synthetic"},Vector:[]float32{.25,.5}}
 for n:=0;n<2;n++{if err:=target.Put(context.Background(),object);err!=nil{t.Fatal(err)}}
 if writes!=2||reads!=0{t.Fatalf("publication performed readback %d %d",writes,reads)}
 actual,err:=target.Get(context.Background(),id);if err!=nil||actual.ID!=id||len(actual.Vector)!=2||reads!=1{t.Fatalf("readback %+v %v",actual,err)}
 fail=true;if err:=target.Put(context.Background(),object);err==nil{t.Fatal("per-object failure accepted")}
}
