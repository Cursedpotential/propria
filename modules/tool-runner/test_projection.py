"""Verify lossless protocol projection and completeness of the hosted Think resource tree."""
import hashlib,json,unittest
import domains
from response_projection import normalize

class ProjectionTests(unittest.TestCase):
    def test_duplicate_native_result_is_flattened(self):
        payload={'data':[{'id':'example','name':'Example workflow'}],'count':1}
        native={'content':[{'type':'text','text':json.dumps(payload)}],'structuredContent':payload,'isError':False}
        self.assertEqual(normalize(native),payload)
    def test_actual_record_and_source_fields_are_not_dropped(self):
        record={'id':'source-1','source_hash':'hash','content':[{'type':'text','text':'{"value":7}'}]}
        self.assertEqual(normalize(record),record)
    def test_nonredundant_metadata_and_text_are_preserved(self):
        native={'content':[{'type':'text','text':'{"value":7}'}],'structuredContent':{'different':8},'isError':False}
        self.assertEqual(normalize(native),native)
        native['_meta']={'source_revision':'1'}
        self.assertEqual(normalize(native),native)
    def test_original_methodology_hashes_match_packaged_bytes(self):
        for entry in domains.THINK:
            digest=hashlib.sha256((domains.LIBRARY/entry['path']).read_bytes()).hexdigest()
            self.assertEqual(digest,entry['sha256'],entry['id'])
    def test_router_support_resources_are_accessible(self):
        value={'access_token':'test-only'}
        for uri in ['think://skills/thinking-model-router/references/decision-matrix.md','think://skills/thinking-model-router/scripts/pick.py','think://skill-registry.json']:
            result=domains.resource(value,'think',uri)
            self.assertGreater(len(result['text']),10)

if __name__=='__main__':unittest.main()
