package weaviate

import (
	"context"
	"errors"
	"encoding/json"
	"net/http"

	"github.com/Cursedpotential/probata/engine/humancontext"
)

// HumanSourceTarget adapts the existing HTTP transport to separately typed native-human objects.
// Inputs: configured existing Store/collection/vector. Outputs: idempotent writes and exact GET readback.
// Effects: object requests only; no collection creation or fabricated PostgreSQL coordinates.
type HumanSourceTarget struct{ Store Store }

// Put upserts one deterministic native-human projection into the configured existing collection.
// Inputs: bounded human object. Outputs: write error or nil. Effects: one search object upsert.
// Choose for native context bundles; PublishObjects retains its legacy PostgreSQL contract.
func(t *HumanSourceTarget)Put(ctx context.Context,object humancontext.SearchObject)error{
	if err:=t.Store.validate();err!=nil{return err};if object.ID==""||len(object.Vector)==0{return errors.New("invalid native human object")}
	body:=map[string]any{"id":object.ID,"class":t.Store.Collection,"properties":object.Properties,"vectors":map[string][]float32{t.Store.VectorName:object.Vector}}
	status,payload,err:=t.Store.do(ctx,http.MethodPost,"/v1/batch/objects",map[string]any{"objects":[]any{body}});if err!=nil{return err};if status!=http.StatusOK{return errors.New("native human search write failed")}
	var results []struct{ID string `json:"id"`;Result struct{Status string `json:"status"`;Errors json.RawMessage `json:"errors"`} `json:"result"`}
	if err:=json.Unmarshal(payload,&results);err!=nil{return err};if len(results)!=1||results[0].ID!=object.ID||results[0].Result.Status!="SUCCESS"||(len(results[0].Result.Errors)>0&&string(results[0].Result.Errors)!="null"){return errors.New("native human search batch rejected object")};return nil
}

// Get reads the complete stored properties and the configured named vector for independent verification.
// Inputs: exact deterministic projection ID. Outputs: typed native object. Effects: one search GET.
// Choose over acknowledged-write counts when finishing the indexing path.
func(t *HumanSourceTarget)Get(ctx context.Context,id string)(humancontext.SearchObject,error){
	object,err:=t.Store.GetObject(ctx,id);if err!=nil{return humancontext.SearchObject{},err}
	if object==nil{return humancontext.SearchObject{},errors.New("native human search object missing")}
	return humancontext.SearchObject{ID:object.ID,Properties:object.Properties,Vector:object.Vectors[t.Store.VectorName]},nil
}
