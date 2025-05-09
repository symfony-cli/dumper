/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

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
			_, _ = out.Write([]byte("\n"))
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
	_, _ = out.Write([]byte("\n"))
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
	_, _ = out.Write([]byte("\n"))
}

// Prints to given output the value(s) that is (are) passed as the argument(s)
// with formatting, indentation and potentially colors
// Pointers are dereferenced.
func defaultDump(values ...interface{}) {
	Fdump(os.Stdout, values...)
}

// Dump points to the default dumper
var Dump = defaultDump
