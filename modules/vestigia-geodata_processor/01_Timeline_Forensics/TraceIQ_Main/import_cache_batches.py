#!/usr/bin/env python3
from __future__ import annotations
import os, sys, json, argparse, re
from pathlib import Path

def _save(p,obj): p.parent.mkdir(parents=True,exist_ok=True); tmp=p.with_suffix(p.suffix+'.tmp'); open(tmp,'w').write(json.dumps(obj)); tmp.replace(p)
def _extract_google(payload):
    if isinstance(payload,dict):
        for pid,rec in payload.items():
            res=rec.get('result') if isinstance(rec,dict) and 'result'in rec else rec
            if isinstance(res,dict): res.setdefault('place_id',res.get('place_id',pid)); yield {'result':res}
        for key in ('results','places','data'):
            if key in payload and isinstance(payload[key],list):
                for it in payload[key]:
                    if isinstance(it,dict): yield {'result': it.get('result') if 'result'in it else it}
    elif isinstance(payload,list):
        for it in payload:
            if isinstance(it,dict): yield {'result': it.get('result') if 'result'in it else it}
def _validate_google(res):
    if not (res.get('place_id') or res.get('id')): return False,'missing_place_id'
    if not res.get('name'): return False,'missing_name'
    if not res.get('formatted_address'): return False,'missing_formatted_address'
    t=res.get('types'); 
    if not t or not isinstance(t,(list,tuple)) or len(t)==0: return False,'missing_types'
    return True,''
def import_google(path, out_dir):
    ok=sk=0; reasons={}
    try: payload=json.load(open(path,'r'))
    except Exception: reasons['read_error']=reasons.get('read_error',0)+1; return ok,sk,reasons
    hit=False
    for rec in _extract_google(payload) or []:
        hit=True; res=rec.get('result') or {}
        v,why=_validate_google(res)
        if not v: sk+=1; reasons[why]=reasons.get(why,0)+1; continue
        pid=res.get('place_id') or res.get('id'); _save(out_dir/f'{pid}.json',{'result':res}); ok+=1
    if not hit: reasons['no_records_found']=reasons.get('no_records_found',0)+1
    return ok,sk,reasons
def _latlng_from_any(obj):
    lv=obj.get('latlng') or obj.get('lat_lng') or obj.get('latlng_key')
    if isinstance(lv,str) and ',' in lv: return lv.strip()
    lat=lng=None
    for k in ('lat','latitude'): 
        if k in obj:
            try: lat=float(obj[k]); break
            except: pass
    for k in ('lng','lon','long','longitude'):
        if k in obj:
            try: lng=float(obj[k]); break
            except: pass
    if lat is not None and lng is not None: return f"{lat},{lng}"
    c=obj.get('coordinate') or obj.get('coordinates') or obj.get('location')
    if isinstance(c,dict):
        try: lat=float(c.get('latitude') or c.get('lat')); lng=float(c.get('longitude') or c.get('lng') or c.get('lon')); return f"{lat},{lng}"
        except: pass
    return None
def _wrap_radar(raw):
    if isinstance(raw,dict):
        if 'addresses'in raw or 'address'in raw: return raw
        if any(k in raw for k in ('formattedAddress','street','city')): return {'address':raw}
    if isinstance(raw,list) and raw and isinstance(raw[0],dict): return {'addresses':raw}
    return {'address':raw}
def _validate_radar(w):
    adr=None
    if 'addresses'in w and isinstance(w['addresses'],list) and w['addresses']: adr=w['addresses'][0]
    elif 'address'in w and isinstance(w['address'],dict): adr=w['address']
    if not isinstance(adr,dict): return False,'missing_address_object'
    if not (adr.get('formattedAddress') or (adr.get('street') and adr.get('city'))): return False,'missing_min_fields'
    return True,''
def import_radar(path, out_dir):
    ok=sk=0; reasons={}; coverage={}
    try: payload=json.load(open(path,'r'))
    except Exception: reasons['read_error']=reasons.get('read_error',0)+1; return ok,sk,reasons,coverage
    def write_one(ll,raw):
        nonlocal ok,sk,reasons,coverage
        w=_wrap_radar(raw); v,why=_validate_radar(w)
        if not v: sk+=1; reasons[why]=reasons.get(why,0)+1; return
        layers=set()
        def add(a): 
            if isinstance(a,dict) and a.get('layer'): layers.add(str(a['layer']))
        if 'addresses'in w and isinstance(w['addresses'],list):
            for a in w['addresses']: add(a)
        elif 'address'in w: add(w['address'])
        coverage[ll]=sorted(set((coverage.get(ll) or [])+list(layers)))
        _save(out_dir/f"{ll.replace(',','_')}.json", w); ok+=1
    if isinstance(payload,dict):
        keys=[k for k in payload.keys() if isinstance(k,str) and (',' in k or re.match(r"^-?\\d+(?:\\.\\d+)?_-?\\d+(?:\\.\\d+)?$",k))]
        if keys:
            for k in keys: write_one(k.replace('_',',') if '_' in k and ',' not in k else k, payload[k])
        else:
            hit=False
            for kk in ('results','addresses','data','items'):
                if kk in payload and isinstance(payload[kk],list):
                    hit=True
                    for it in payload[kk]:
                        if isinstance(it,dict):
                            ll=_latlng_from_any(it) or _latlng_from_any(payload)
                            if ll: write_one(ll,it)
                    break
            if not hit:
                ll=_latlng_from_any(payload)
                if ll: write_one(ll,payload)
    elif isinstance(payload,list):
        for it in payload:
            if isinstance(it,dict):
                ll=_latlng_from_any(it) or _latlng_from_any(payload)
                if ll: write_one(ll,it)
    return ok,sk,reasons,coverage
def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--google",action="append",default=[])
    ap.add_argument("--radar",action="append",default=[])
    ap.add_argument("--out-google",default="raw_api/google/place_details")
    ap.add_argument("--out-radar", default="raw_api/radar/reverse_geocode")
    args=ap.parse_args()
    G=Path(os.path.expanduser(args.out_google)); R=Path(os.path.expanduser(args.out_radar))
    G.mkdir(parents=True, exist_ok=True); R.mkdir(parents=True, exist_ok=True)
    for g in args.google:
        ok,sk,rs=import_google(Path(g), G); Path(f"{g}.import_report.json").write_text(json.dumps({"provider":"google","wrote":ok,"skipped":sk,"reasons":rs}, indent=2))
        print(f"[google] {g}: wrote {ok}, skipped {sk}")
    for r in args.radar:
        ok,sk,rs,cov=import_radar(Path(r), R); Path(f"{r}.import_report.json").write_text(json.dumps({"provider":"radar","wrote":ok,"skipped":sk,"reasons":rs,"layer_coverage":cov}, indent=2))
        print(f"[radar ] {r}: wrote {ok}, skipped {sk}")
    print("[done]")
if __name__=="__main__": main()
