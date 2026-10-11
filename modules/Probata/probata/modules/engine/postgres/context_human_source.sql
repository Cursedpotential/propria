-- Extract native SMS Backup & Restore fields from bounded generic XML lexical rows.
-- Inputs: $lexemes (bounded lexical JSON), $source_ref, $provider_version_id, $source_version_ref.
-- Outputs: ordered human-source native JSON and original byte-range locators; no table writes.
-- Side effects: none; DuckDB reads bounded lexical JSON and original attachment payload remains in place.
-- Pick after context registration/framing, without legacy retained-object or handler prerequisites.
-- Native field/clock authority: https://www.synctech.com.au/sms-backup-restore/fields-in-xml-backup-files/
-- Synthetic <v a=.../> wrappers below decode escaped attribute bytes only; they are not original bytes.
WITH lexical AS (
    SELECT unnest(from_json($lexemes, '[{"kind":"VARCHAR","element_id":"BIGINT","parent_id":"BIGINT","record_ordinal":"BIGINT",
      "name":"VARCHAR","start":"BIGINT","end":"BIGINT","value_start":"BIGINT","value_end":"BIGINT",
      "value_bytes":"BIGINT","raw_value_b64":"VARCHAR","quote":"VARCHAR","range_only":"BOOLEAN","self_closing":"BOOLEAN"}]'), recursive:=true)
), decoded AS (
    SELECT *, CASE WHEN NOT coalesce(range_only,false)
        THEN TRY(xml_extract_text('<v a=' || quote || decode(from_base64(coalesce(raw_value_b64,''))) || quote || '/>', '/v/@a')[1])
        END AS native_value
    FROM lexical WHERE kind='attribute'
), attributes AS (
    SELECT element_id,
      max(native_value) FILTER(WHERE name='address') AS address,
      max(native_value) FILTER(WHERE name='type') AS native_type,
      max(native_value) FILTER(WHERE name='msg_box') AS msg_box,
      max(native_value) FILTER(WHERE name='date') AS native_date,
      max(native_value) FILTER(WHERE name='body') AS body,
      max(native_value) FILTER(WHERE name='text') AS text_value,
      max(native_value) FILTER(WHERE name='ct') AS content_type,
      coalesce(max(native_value) FILTER(WHERE name='name'),max(native_value) FILTER(WHERE name='cl')) AS filename,
      list(struct_pack(name:=name,value:=CASE WHEN name='data' THEN NULL ELSE native_value END,
        value_start:=value_start,value_end:=value_end,value_bytes:=coalesce(value_bytes,0),
        range_only:=coalesce(range_only,false),quote:=quote) ORDER BY start) AS native_attributes,
      count(*) != count(DISTINCT name) AS duplicate_attributes,
      bool_or((NOT coalesce(range_only,false) AND native_value IS NULL)
        OR (coalesce(range_only,false) AND name!='data')) AS unsupported_value
    FROM decoded GROUP BY element_id
), elements AS (
    SELECT l.*,regexp_replace(l.name,'^.*:','') AS local_name,
      CASE WHEN coalesce(l.self_closing,false) THEN l."end"
        ELSE (SELECT max(e."end") FROM lexical e WHERE e.kind='element_end' AND e.element_id=l.element_id) END AS element_end
    FROM lexical l WHERE kind='element_start'
), records AS (
    SELECT e.*,a.address,a.native_type,a.msg_box,a.native_date,a.body,a.native_attributes,
      a.duplicate_attributes,a.unsupported_value
    FROM elements e LEFT JOIN attributes a USING(element_id)
    WHERE e.parent_id IN (SELECT element_id FROM elements WHERE parent_id=0)
), parts AS (
    SELECT r.element_id AS record_id,e.element_id,e.start,a.text_value,a.content_type,a.filename,a.native_attributes,
      a.duplicate_attributes,a.unsupported_value,
      (SELECT list(struct_pack(source_ref:=$source_ref,provider_version_id:=$provider_version_id,
           source_version_ref:=$source_version_ref,start:=d.value_start,"end":=d.value_end,
           original_encoding:='xml_attribute_base64',payload_status:='original_range_not_materialized') ORDER BY d.start)
       FROM decoded d WHERE d.element_id=e.element_id AND d.name='data') AS payload_ranges
    FROM records r JOIN elements group_element ON group_element.parent_id=r.element_id AND group_element.local_name='parts'
    JOIN elements e ON e.parent_id=group_element.element_id AND e.local_name='part'
    LEFT JOIN attributes a ON a.element_id=e.element_id
), native_children AS (
    SELECT record_ordinal,list(struct_pack(name:=name,element_id:=element_id,parent_id:=parent_id,
        start:=start,"end":=element_end,native_attributes:=a.native_attributes) ORDER BY start) AS elements,
        bool_or(local_name NOT IN ('sms','mms','parts','part','addrs','addr')) AS unsupported_child
    FROM elements e LEFT JOIN attributes a USING(element_id)
    WHERE record_ordinal>0 GROUP BY record_ordinal
), part_groups AS (
    SELECT record_id,
      string_agg(text_value, chr(10) ORDER BY start) FILTER(WHERE content_type='text/plain' AND text_value IS NOT NULL AND text_value!='null') AS text_body,
      list(struct_pack(element_id:=element_id,content_type:=content_type,filename:=filename,native_attributes:=native_attributes,payload_ranges:=payload_ranges) ORDER BY start) AS native_parts,
      bool_or(coalesce(duplicate_attributes,false) OR coalesce(unsupported_value,false)) AS unsupported_part
    FROM parts GROUP BY record_id
), addresses AS (
    SELECT r.element_id AS record_id,e.start,a.address,a.native_type,a.native_attributes,
      a.duplicate_attributes,a.unsupported_value
    FROM records r JOIN elements group_element ON group_element.parent_id=r.element_id AND group_element.local_name='addrs'
    JOIN elements e ON e.parent_id=group_element.element_id AND e.local_name='addr'
    LEFT JOIN attributes a ON a.element_id=e.element_id
), address_groups AS (
    SELECT record_id,
      list(struct_pack(address:=address,native_type:=native_type,native_attributes:=native_attributes) ORDER BY start) AS participants,
      list(address ORDER BY start) FILTER(WHERE native_type='137') AS senders,
      list(address ORDER BY start) FILTER(WHERE native_type IN ('151','130','129')) AS recipients,
      bool_or(coalesce(duplicate_attributes,false) OR coalesce(unsupported_value,false)) AS unsupported_address
    FROM addresses GROUP BY record_id
)
SELECT r.record_ordinal,
    CASE WHEN r.local_name NOT IN ('sms','mms') THEN 'unparsed'
         WHEN r.element_end IS NULL OR coalesce(r.duplicate_attributes,false) OR coalesce(r.unsupported_value,false)
           OR coalesce(p.unsupported_part,false) OR coalesce(a.unsupported_address,false)
           OR coalesce(c.unsupported_child,false)
           OR EXISTS(SELECT 1 FROM lexical content WHERE content.record_ordinal=r.record_ordinal AND content.kind IN ('text','cdata')) THEN 'pending'
         ELSE 'complete' END AS extraction_status,
    json_object('record_kind',CASE WHEN r.local_name IN ('sms','mms') THEN 'message' ELSE 'object' END,'message_kind',r.local_name,
      'body',CASE WHEN r.local_name='sms' THEN nullif(r.body,'null') ELSE p.text_body END,
      'direction',CASE WHEN r.local_name='sms' THEN CASE r.native_type WHEN '1' THEN 'received' WHEN '2' THEN 'sent' ELSE 'unknown' END
        ELSE CASE r.msg_box WHEN '1' THEN 'received' WHEN '2' THEN 'sent' ELSE 'unknown' END END,
      'native_date',r.native_date,
      'occurred_at',TRY(strftime(epoch_ms(try_cast(r.native_date AS BIGINT)),'%Y-%m-%dT%H:%M:%S.%fZ')),
      'native_attributes',r.native_attributes,'native_parts',p.native_parts,'native_elements',c.elements,
      'native_content_ranges',(SELECT to_json(list(struct_pack(kind:=content.kind,start:=content.start,"end":=content."end") ORDER BY content.start)) FROM lexical content WHERE content.record_ordinal=r.record_ordinal AND content.kind IN ('text','cdata')),
      'participants',CASE WHEN r.local_name='sms' THEN to_json([r.address]) ELSE to_json(a.participants) END,
      'senders',a.senders,'recipients',a.recipients,
      'sender_address',CASE WHEN r.local_name='sms' AND r.native_type='1' THEN r.address ELSE a.senders[1] END,
      'sender_role',CASE WHEN r.local_name='sms' AND r.native_type='2' THEN 'source_owner' ELSE NULL END,
      'source_locator',json_object('source_ref',$source_ref,'provider_version_id',$provider_version_id,
         'source_version_ref',$source_version_ref,'record_ordinal',r.record_ordinal,'start',r.start,'end',r.element_end)
    )::VARCHAR AS native_fields
FROM records r LEFT JOIN part_groups p ON p.record_id=r.element_id
LEFT JOIN address_groups a ON a.record_id=r.element_id
LEFT JOIN native_children c ON c.record_ordinal=r.record_ordinal
ORDER BY r.record_ordinal
