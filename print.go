package dumper

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
)

const ElementsPerLine = 30

func (s *state) WithTempBuffer(fn func(buf *bytes.Buffer)) string {
	buf := &bytes.Buffer{}
	var previousBuf io.Writer

	previousBuf, s.w = s.w, buf
	fn(buf)

	s.w = previousBuf

	return buf.String()
}

func (s *state) print(args ...interface{}) {
	fmt.Fprint(s, args...)
}

func (s *state) printf(format string, args ...interface{}) {
	fmt.Fprintf(s, format, args...)
}

func (s *state) Pad() {
	s.Write(bytes.Repeat([]byte("  "), s.depth))
}

func (s *state) breakLineIfNecessary(n, i int) bool {
	if mod := i % ElementsPerLine; mod == 0 || s.forceNewLines {
		if n > ElementsPerLine || s.forceNewLines {
			s.printf("\n")
			s.Pad()
		}

		return false
	}

	return true
}

func (s *state) printfStyle(typ string, format string, v ...interface{}) {
	if style := s.styles[typ]; style != "" {
		format = fmt.Sprintf("\033[%sm%s\033[m", style, format)
	}
	s.printf(format, v...)
}

func (s *state) ForceNewLines(v bool) bool {
	v, s.forceNewLines = s.forceNewLines, v

	return v
}

func (s *state) DumpString(str string) {
	s.printfStyle("const", "\"")
	s.printfStyle("str", "%v", str)
	s.printfStyle("const", "\"")
}

func (s *state) DumpScalar(v interface{}, t reflect.Type, dumpTypeInstantiation bool) {
	if (s.forceDumpTypeInstantiation || s.depth == 0) && dumpTypeInstantiation {
		s.printfStyle("meta", "%v", t.Name())
		s.print("(")
		s.printfStyle("num", "%v", v)
		s.print(")")
	} else {
		s.printfStyle("num", "%v", v)
	}
}

func (s *state) DumpComplex(v complex128, t reflect.Kind) {
	str := s.WithTempBuffer(func(buf *bytes.Buffer) {
		r, i := real(v), imag(v)

		if i != 0 {
			s.printfStyle("meta", "complex")
			s.printf("(%v, %v)", r, i)
		} else {
			s.printf("%v", r)
		}
	})

	if t == reflect.Complex64 {
		s.printf("complex64(%s)", str)
	} else {
		s.print(str)
	}
}

func (s *state) DumpStructType(t reflect.Type) {
	name := t.Name()
	if name == "" {
		name = t.String()
		s.AddComment("anonymous struct")
	} else {
		name = fmt.Sprintf("%s.%s", filepath.Base(t.PkgPath()), name)
	}
	s.printfStyle("note", "%v", name)
}

func (s *state) DumpStructComments(v reflect.Value) string {
	str := s.WithTempBuffer(func(buf *bytes.Buffer) {
		switch v.Kind() {
		// reflect.Int and reflect.Float64 don't need comments
		// neither reflect.Complex64 and reflect.Complex128 as they required instantiation
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32:
			s.printfStyle("meta", "%v", v.Type().Name())

		case reflect.Ptr:
			if v.IsNil() {
				if s.currentPointerName != "" {
					s.printfStyle("ref", "%v ", s.currentPointerName)
				}
				s.currentPointerName = ""
				s.printfStyle("meta", "&%v", v.Type().Elem())
			}
		}
	})

	s.prependComment(str)

	if len(s.comments) == 0 {
		return ""
	}

	defer s.ResetComments()
	return s.formatComments()
}

func DumpStructWithPrivateFields(s State, v reflect.Value) {
	hidePrivateFields := false
	if ss, ok := s.(*state); ok {
		ss.DumpStructFields(v, &hidePrivateFields)
	}
}

func (s *state) DumpStructFields(value reflect.Value, hidePrivateFields *bool) {
	typ := value.Type()

	for i, numFields := 0, value.NumField(); i < numFields; i++ {
		field := typ.Field(i)
		// this is an unexported field
		if field.PkgPath != "" {
			// Hide private field for external packages
			if hidePrivateFields == nil {
				if field.PkgPath != s.lastCaller {
					continue
				}
			} else if *hidePrivateFields {
				continue
			}
		}
		s.DumpStructField(field.Name, value.Field(i))
	}
}

func (s *state) DumpStructField(fieldName string, v reflect.Value) {
	s.Pad()
	s.printf("%v: ", fieldName)
	s.dumpVal(v)
	s.printf(",%s\n", s.DumpStructComments(v))
}

type mapKeysSorter struct {
	keys []reflect.Value
}

func (s mapKeysSorter) Len() int {
	return len(s.keys)
}

func (s mapKeysSorter) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}

func (s mapKeysSorter) Less(i, j int) bool {
	return s.keys[i].String() < s.keys[j].String()
}
