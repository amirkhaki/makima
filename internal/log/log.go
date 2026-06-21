package log

import (
	"fmt"
	"os"
	"strings"
)

var verbose bool

func SetVerbose(v bool) {
	verbose = v
}

func IsVerbose() bool {
	return verbose
}

func Info(msg string, args ...any) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "[makima] "+msg+"\n", args...)
}

func Event(source, msg string, args ...any) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "[makima:%s] "+msg+"\n", append([]any{source}, args...)...)
}

func Debug(source, msg string, args ...any) {
	if !verbose {
		return
	}
	fmt.Fprintf(os.Stderr, "[makima:%s:debug] "+msg+"\n", append([]any{source}, args...)...)
}

func Error(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "[makima:error] "+msg+"\n", args...)
}

func Raw(msg string) {
	if !verbose {
		return
	}
	fmt.Fprintln(os.Stderr, msg)
}

func Dump(label string, data map[string]string) {
	if !verbose {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[makima] %s:\n", label))
	for k, v := range data {
		sb.WriteString(fmt.Sprintf("  %s = %s\n", k, v))
	}
	fmt.Fprint(os.Stderr, sb.String())
}
