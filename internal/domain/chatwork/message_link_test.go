package chatwork

import (
	"strings"
	"testing"
)

func TestParseMessageLinkExactReferences(t *testing.T) {
	for _, pair := range [][2]string{{"42", "101"}, {strings.Repeat("9", 32), strings.Repeat("8", 32)}} {
		got, err := ParseMessageLink(messageLinkPrefix + pair[0] + "-" + pair[1])
		if err != nil || got.Room.Value != pair[0] || got.Message.Value != pair[1] || got.Room.Kind != ReferenceRoom || got.Message.Kind != ReferenceMessage {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	}
}

func TestParseMessageLinkRejectsNoncanonicalAndUntrustedInput(t *testing.T) {
	for _, link := range []string{
		"", messageLinkPrefix + "42", messageLinkPrefix + "0-1", messageLinkPrefix + "01-1", messageLinkPrefix + "1-01", messageLinkPrefix + "1-0", messageLinkPrefix + strings.Repeat("1", 33) + "-1",
		messageLinkPrefix + "1-2-3", messageLinkPrefix + "1-2?x", messageLinkPrefix + "1-2\n", messageLinkPrefix + "1-2%20", messageLinkPrefix + "１-2",
		"http://www.chatwork.com/#!rid1-2", "https://chatwork.com/#!rid1-2", "https://www.chatwork.com:443/#!rid1-2", "https://user@www.chatwork.com/#!rid1-2", "https://www.chatwork.com/other#!rid1-2", "https://www.chatwork.com/?x=1#!rid1-2", "https://www.chatwork.com/#!RID1-2",
	} {
		if got, err := ParseMessageLink(link); err == nil || got != (MessageLinkReferences{}) {
			t.Errorf("accepted invalid form, result=%+v", got)
		}
	}
}
