# Byline: Claude Code · Opus 5.5 · 2026-10-02
# Device own number per SMS Backup & Restore file, for the casevault device folder (owner 2026-10-02 07:35:
# the folder is the device's own 10-digit number). Evidence, in order: MMS addr type 137 (from) in the sent
# box (msg_box 2); fallback MMS addr type 151 (to) in the received box. Calls files carry no own number, so
# only their root backup_set is read (a calls file pairs with the SMS file of the same backup_set).
# Streams each object once with rclone cat on ovh-files (EnvironmentFile for B2), stops early once settled.
# Read-only. Usage: python3 sms_backup_own_number.py <plan.tsv> <out.tsv>
# out.tsv: key, root tag, count, backup_set, backup_date, mms_seen, sent_from top3, recv_to top4, parse error
import collections
import re
import subprocess
import sys
import xml.etree.ElementTree as ET

CONF = "/opt/casebible/rclone.conf"


def top(counter, n):
    return ",".join(f"{k}:{v}" for k, v in counter.most_common(n))


def main(plan_path, out_path):
    with open(out_path, "w") as out:
        for line in open(plan_path):
            src_root, _dst_root, rel = line.rstrip("\n").split("\t")
            if not rel.endswith(".xml"):
                continue
            key = src_root + rel
            proc = subprocess.Popen(["rclone", "--config", CONF, "cat", "b2:salem-data/" + key], stdout=subprocess.PIPE)
            sent, recv, root, seen, err = collections.Counter(), collections.Counter(), {}, 0, ""
            try:
                for event, el in ET.iterparse(proc.stdout, events=("start", "end")):
                    if event == "start":
                        if el.tag in ("smses", "calls") and not root:
                            root = dict(el.attrib)
                            root["tag"] = el.tag
                        if el.tag == "calls":
                            break
                        continue
                    if el.tag == "mms":
                        seen += 1
                        box = el.get("msg_box")
                        for addr in el.iter("addr"):
                            number = re.sub(r"\D", "", addr.get("address") or "")[-10:]
                            if len(number) != 10:
                                continue
                            if box == "2" and addr.get("type") == "137":
                                sent[number] += 1
                            if box == "1" and addr.get("type") == "151":
                                recv[number] += 1
                        el.clear()
                        if sum(sent.values()) >= 50:
                            break
                    elif el.tag == "sms":
                        el.clear()
            except ET.ParseError as error:
                err = str(error)[:120]
            proc.kill()
            out.write("\t".join([key, root.get("tag", ""), root.get("count", "") or "", root.get("backup_set", "") or "",
                                 root.get("backup_date", "") or "", str(seen), top(sent, 3), top(recv, 4), err]) + "\n")
            out.flush()


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
