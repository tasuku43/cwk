package capsule

import (
	"fmt"
	"strings"
	"time"
)

const fixedJSTOffsetSeconds = 9 * 60 * 60

var (
	fixedJSTLocation          = time.FixedZone("JST", fixedJSTOffsetSeconds)
	minimumJSTRFC3339UnixTime = time.Date(0, time.January, 1, 0, 0, 0, 0, fixedJSTLocation).Unix()
	maximumJSTRFC3339UnixTime = time.Date(10000, time.January, 1, 0, 0, 0, 0, fixedJSTLocation).Unix()
)

func formatSendTimeJST(unixSeconds int64) string {
	if unixSeconds < minimumJSTRFC3339UnixTime || unixSeconds >= maximumJSTRFC3339UnixTime {
		return "unavailable"
	}
	return time.Unix(unixSeconds, 0).In(fixedJSTLocation).Format(time.RFC3339)
}

func appendSendTimeJST(output *strings.Builder, unixSeconds int64, enabled bool) {
	if enabled {
		fmt.Fprintf(output, " send-time-jst=%s", formatSendTimeJST(unixSeconds))
	}
}

func messageListSchema(limitMessageBodies, includeSendTimeJST bool) string {
	var schema strings.Builder
	schema.WriteString(`schema: #sequence message-ref actor sent`)
	if includeSendTimeJST {
		schema.WriteString(` [send-time-jst]`)
	}
	schema.WriteString(` [reply] [to] [quote] [quote-candidate] [relation-state]`)
	if limitMessageBodies {
		schema.WriteString(` [body-truncated=true original-body-bytes=N]`)
	}
	schema.WriteString(` "body"`)
	return schema.String()
}
