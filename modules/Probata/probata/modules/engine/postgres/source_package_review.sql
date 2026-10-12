-- Extend existing correction history to native registered sources without fake custody pins.
-- Inputs: DBA-reviewed platform schema. Outputs: nullable legacy coordinates for native rows.
-- Effects: additive admission/check/index changes only; no runtime applies DDL automatically.
-- Pick before releasing package review; existing retained-preview correction rules stay enforced.
BEGIN;
ALTER TABLE context.source_metadata_correction ALTER COLUMN subject_sha256 DROP NOT NULL;
ALTER TABLE context.source_metadata_correction ALTER COLUMN preview_handle DROP NOT NULL;
ALTER TABLE context.source_metadata_correction ADD CONSTRAINT source_metadata_correction_native_coordinate_check
 CHECK ((subject_sha256 IS NOT NULL AND preview_handle IS NOT NULL)
 OR (subject_sha256 IS NULL AND preview_handle IS NULL AND field_key LIKE 'package:%'));
CREATE UNIQUE INDEX source_metadata_correction_native_revision
 ON context.source_metadata_correction(source_version_id,field_key,revision)
 WHERE subject_sha256 IS NULL;
ALTER TABLE context.source_metadata_correction ADD CONSTRAINT source_metadata_correction_native_scope
 UNIQUE(correction_ref,source_version_id,field_key);
ALTER TABLE context.source_metadata_correction ADD CONSTRAINT source_metadata_correction_native_supersedes_fk
 FOREIGN KEY(supersedes_ref,source_version_id,field_key)
 REFERENCES context.source_metadata_correction(correction_ref,source_version_id,field_key);
-- Reuse the existing BFF reader role for lossless registered package inspection.
GRANT SELECT ON context.source_metadata,context.source_metadata_correction,
 context.source_version_object,context.retained_object,context.proffer_preview_binding
 TO workbench_reader;
COMMIT;
