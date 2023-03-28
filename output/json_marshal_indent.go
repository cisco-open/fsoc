package output

import (
	"os"
	"strconv"
	"strings"
)

var indent_spaces = 2 // Default value

func init() {
	if s := os.Getenv("FSOC_JSON_INDENT"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			indent_spaces = n
		}
	}
}

func GetJsonIndent() string {
	// return empty space repeated json_indent times
	return strings.Repeat(" ", indent_spaces)

}
