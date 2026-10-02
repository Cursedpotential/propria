import glob,sys,re
sys.path.insert(0,'.')
import oracle
n=0; naive_changed=0; sigs=0; nonascii=0
for f in sorted(glob.glob("../samples/**/message_*.html",recursive=True)):
    for b in oracle.fb_blocks(open(f,encoding="utf-8").read()):
        for t in [b["sender"],b["body"]]+b["reactions"]:
            n+=1
            if re.search(r"[^\x00-\x7f]",t): nonascii+=1
            if re.search("[ÃÂâð][\u0080-¿]",t): sigs+=1
            try:
                fixed=t.encode("latin-1").decode("utf-8")
                if fixed!=t: naive_changed+=1
            except (UnicodeEncodeError,UnicodeDecodeError): pass
print({"strings":n,"with_non_ascii":nonascii,"mojibake_signature_strings":sigs,"changed_by_latin1_utf8_roundtrip":naive_changed})
