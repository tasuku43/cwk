package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseRoomTaskDeadline preserves the existing Unix-seconds input and converts
// the optional RFC3339 spelling at the CLI boundary. The domain and provider
// contracts continue to carry only positive Unix seconds.
func parseRoomTaskDeadline(value string) (int64, error) {
	if unixSeconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if unixSeconds > 0 {
			return unixSeconds, nil
		}
		return 0, invalidRoomTaskDeadline()
	}

	if !isWholeSecondRFC3339(value) || strings.HasSuffix(value, "-00:00") {
		return 0, invalidRoomTaskDeadline()
	}
	deadline, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0, invalidRoomTaskDeadline()
	}
	unixSeconds := deadline.Unix()
	if unixSeconds <= 0 {
		return 0, invalidRoomTaskDeadline()
	}
	return unixSeconds, nil
}

func isWholeSecondRFC3339(value string) bool {
	if len(value) == len("2006-01-02T15:04:05Z") {
		return value[19] == 'Z'
	}
	if len(value) != len("2006-01-02T15:04:05+07:00") {
		return false
	}
	if value[19] != '+' && value[19] != '-' {
		return false
	}
	if value[22] != ':' {
		return false
	}
	if !asciiDigits(value[20:22]) || !asciiDigits(value[23:25]) {
		return false
	}
	offsetHours, errHours := strconv.Atoi(value[20:22])
	offsetMinutes, errMinutes := strconv.Atoi(value[23:25])
	return errHours == nil && errMinutes == nil && offsetHours <= 23 && offsetMinutes <= 59
}

func asciiDigits(value string) bool {
	for index := range value {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func invalidRoomTaskDeadline() error {
	return fmt.Errorf("--limit は正のUnix秒または秒精度RFC3339（明示タイムゾーン付き）で指定してください")
}
