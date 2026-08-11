package chatwork

import (
	"strings"
	"testing"
)

func TestValidateMessageBodyPreservesBoundedUTF8Text(t *testing.T) {
	for _, body := range []string{
		"[info][title]確認[/title]\n$HOME $(date) > result\n[/info]",
		strings.Repeat("x", MaxMessageBodyBytes),
	} {
		if err := ValidateMessageBody(body); err != nil {
			t.Fatalf("body length %d: %v", len(body), err)
		}
	}
}

func TestValidateMessageBodyRejectsInvalidContentWithoutEchoingIt(t *testing.T) {
	secret := "private-message-fragment"
	for name, body := range map[string]string{
		"empty":        "",
		"too large":    strings.Repeat("x", MaxMessageBodyBytes+1),
		"invalid UTF8": string([]byte{0xff, 0xfe}),
		"NUL":          secret + "\x00",
	} {
		t.Run(name, func(t *testing.T) {
			err := ValidateMessageBody(body)
			if err == nil {
				t.Fatal("invalid body succeeded")
			}
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("error exposed message content: %v", err)
			}
		})
	}
}

func TestMessageMutationRequestRequiresExactlyOneBodySource(t *testing.T) {
	for _, task := range []Task{TaskMessagesSend, TaskMessagesUpdate} {
		for name, request := range map[string]Request{
			"literal": {Task: task, Body: "message"},
			"file":    {Task: task, BodyFile: "message.txt"},
		} {
			t.Run(string(task)+"/"+name, func(t *testing.T) {
				if err := request.Validate(); err != nil {
					t.Fatal(err)
				}
			})
		}
		for name, request := range map[string]Request{
			"missing": {Task: task},
			"both":    {Task: task, Body: "message", BodyFile: "message.txt"},
		} {
			t.Run(string(task)+"/"+name, func(t *testing.T) {
				if err := request.Validate(); err == nil {
					t.Fatal("invalid body-source combination succeeded")
				}
			})
		}
	}

	if err := (Request{Task: TaskRoomsList, BodyFile: "message.txt"}).Validate(); err == nil {
		t.Fatal("body file succeeded for a non-message task")
	}
}
