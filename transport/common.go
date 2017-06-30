package transport

import (
	"encoding/base64"
	"fmt"
	"regexp"
)

var safeRe, safeReErr = regexp.Compile("^[^']+$")

func init() {
	if safeReErr != nil {
		panic(safeReErr)
	}
}

func escape(in string) string {
	if safeRe.MatchString(in) {
		return fmt.Sprintf("'%s'", in)
	}

	b64 := base64.StdEncoding.EncodeToString([]byte(in))
	return fmt.Sprintf("$(echo '%s' | base64 -w0 -d)", b64)
}
