package dumper

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

func lastCaller() string {
	var pcs [5]uintptr
	runtime.Callers(2, pcs[:])
	lastCaller := ""

	for _, pc := range pcs {
		fn := runtime.FuncForPC(pc - 1)
		if fn == nil {
			return ""
		}
		name := fn.Name()
		if strings.HasPrefix(name, "runtime") {
			break
		}
		lastPackage := filepath.Base(name)
		if pos := strings.Index(lastPackage, "."); pos != -1 {
			lastPackage = lastPackage[:pos]
		}
		lastCaller = filepath.Join(filepath.Dir(name), lastPackage)
		if strings.Contains(name, "/dumper") && !strings.Contains(name, ".(*DumperSuite)") {
			continue
		} else if strings.Contains(name, "/console.Dump") {
			continue
		}

		break
	}

	return lastCaller
}

func fdump(out io.Writer, styles map[string]string, values ...interface{}) {
	for i, value := range values {
		if i > 0 {
			out.Write([]byte("\n"))
		}
		state := state{
			styles:     styles,
			pointers:   mapPointers(reflect.ValueOf(value)),
			comments:   []string{},
			w:          out,
			lastCaller: lastCaller(),
		}
		state.Dump(value)
	}
}

// Fdump prints to the writer the value with indentation.
func Fdump(out io.Writer, values ...interface{}) {
	fdump(out, defaultStyles, values...)
	out.Write([]byte("\n"))
}

// Sdump dumps the values into a string with indentation.
func Sdump(values ...interface{}) string {
	buf := &bytes.Buffer{}
	buf.Reset()

	fdump(buf, defaultStyles, values...)

	return buf.String()
}

// FdumpColor prints to the writer the value with indentation and color.
func FdumpColor(out io.Writer, values ...interface{}) {
	fdump(out, colorStyles, values...)
	out.Write([]byte("\n"))
}

// Prints to given output the value(s) that is (are) passed as the argument(s)
// with formatting, indentation and potentially colors
// Pointers are dereferenced.
func defaultDump(values ...interface{}) {
	Fdump(os.Stdout, values...)
}

// Dump points to the default dumper
var Dump = defaultDump
