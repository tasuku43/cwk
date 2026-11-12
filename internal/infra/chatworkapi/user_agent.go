package chatworkapi

import (
	"regexp"
	"strings"
)

const maxUserAgentVersionBytes = 128

var userAgentVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-([0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*))?$`)

func chatworkUserAgent(version string) string {
	if !validChatworkVersion(version) {
		version = "dev"
	}
	return "cwk/" + version
}

func validChatworkVersion(version string) bool {
	if version == "dev" {
		return true
	}
	if len(version) == 0 || len(version) > maxUserAgentVersionBytes {
		return false
	}
	match := userAgentVersionPattern.FindStringSubmatch(version)
	if match == nil {
		return false
	}
	for _, identifier := range strings.Split(match[5], ".") {
		if numericIdentifier(identifier) && len(identifier) > 1 && identifier[0] == '0' {
			return false
		}
	}
	return true
}

func numericIdentifier(identifier string) bool {
	if identifier == "" {
		return false
	}
	for index := 0; index < len(identifier); index++ {
		if identifier[index] < '0' || identifier[index] > '9' {
			return false
		}
	}
	return true
}
