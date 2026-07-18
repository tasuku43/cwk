package chatworkapi

import "testing"

func TestParseNotationKeepsOnlyExplicitRelations(t *testing.T) {
	body := "[To:12] A [rp aid=13 to=7-42] B [qt][qtmeta aid=14 time=1700000000][To:99][/qt] [code][To:98][/code]"
	recipients, reply, quotes, err := parseNotation(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(recipients) != 2 || recipients[0].Value != "12" || recipients[1].Value != "13" {
		t.Fatalf("recipients = %+v", recipients)
	}
	if reply == nil || reply.Target.Value != "42" || reply.ExternalID != "7" || reply.Resolved {
		t.Fatalf("reply = %+v", reply)
	}
	if len(quotes) != 1 || quotes[0].Target.Value != "14" || quotes[0].ExternalID != "1700000000" || quotes[0].Resolved {
		t.Fatalf("quotes = %+v", quotes)
	}
}

func TestParseNotationRejectsMalformedRecognizedTags(t *testing.T) {
	for _, body := range []string{"[To:abc]", "[rp aid=1 to=2-x]", "[qt][qtmeta aid=1]missing close", "[code]missing close", "[rp aid=1 to=2-3][rp aid=1 to=2-4]"} {
		if _, _, _, err := parseNotation(body); err == nil {
			t.Fatalf("parseNotation(%q) succeeded", body)
		}
	}
}
