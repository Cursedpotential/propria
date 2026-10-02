// Byline: Claude Code · Opus 5.5 · 2026-10-02
package firstparty

import "testing"

func TestPlatformForDetectedFormatNamesOnlyFacebookMessenger(t *testing.T) {
	platform, capture, representation, ok := PlatformForDetectedFormat("facebook_messenger_json")
	if !ok || platform != "facebook_messenger" || capture != "facebook_export" || representation != "json" {
		t.Fatalf("facebook_messenger_json = %q %q %q %v", platform, capture, representation, ok)
	}
	for _, format := range []string{"json", "ndjson", "smsbackuprestore_xml", "chatgpt_official_json", ""} {
		if _, _, _, ok := PlatformForDetectedFormat(format); ok {
			t.Fatalf("%q must not resolve a platform from its signature", format)
		}
	}
}
