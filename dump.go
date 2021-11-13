package dumper

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
)

type state struct {
	w io.Writer

	styles   map[string]string
	comments []string

	depth                      int
	forceDumpTypeInstantiation bool
	forceNewLines              bool

	pointers           visitedPointersMap
	currentPointerName string

	lastCaller string
}

func (s *state) Write(p []byte) (n int, err error) {
	return s.w.Write(p)
}

func (s *state) AddComment(comment string) {
	if len(comment) == 0 {
		return
	}

	s.comments = append(s.comments, comment)
}

func (s *state) prependComment(comment string) {
	if len(comment) == 0 {
		return
	}

	s.comments = append([]string{comment}, s.comments...)
}

func (s *state) ResetComments() (ret []string) {
	ret, s.comments = s.comments, []string{}

	return
}

func (s *state) formatComments() string {
	return fmt.Sprintf(" // %s", strings.Join(s.comments, ", "))
}

func (s *state) Dump(value interface{}) {
	v := reflect.ValueOf(value)
	s.dumpVal(v)
}

func (s *state) DepthUp() {
	s.depth--
}

func (s *state) DepthDown() {
	s.depth++
}

func (s *state) dumpVal(value reflect.Value) {
	if s.handleCircularRef(value) {
		return
	}

	kind := value.Kind()

	if kind == reflect.Invalid {
		// Do nothing
		s.printf("<invalid>")
		return
	}

	typ := value.Type()

	// Handle custom dumpers
	if typ.Implements(dumpableType) {
		s.dumpCustom(value, value.Interface().(Dumpable))
		return
	}

	for t, dumper := range customDumpers {
		if t == typ {
			s.dumpCustomFn(value, dumper)
			return
		}
	}

	switch kind {

	case reflect.Bool:
		s.printfStyle("const", "%t", value.Bool())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		s.DumpScalar(value.Int(), typ, kind != reflect.Int)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s.DumpScalar(value.Uint(), typ, true)

	case reflect.Float32, reflect.Float64:
		s.DumpScalar(value.Float(), typ, kind != reflect.Float64)

	case reflect.Complex64, reflect.Complex128:
		s.DumpComplex(value.Complex(), kind)

	case reflect.String:
		s.DumpString(value.String())

	case reflect.UnsafePointer:
		s.print("unsafe.Pointer(")
		s.Dump(value.Pointer())
		s.print(")")

	case reflect.Uintptr:
		s.printf("%s(0x%x)", typ.Name(), value.Uint())

	case reflect.Ptr:
		if value.IsNil() {
			s.printfStyle("ref", "nil")

			if s.depth == 0 {
				s.print(s.DumpStructComments(value))
			}
		} else {
			previousComments := s.ResetComments()

			s.AddComment(s.WithTempBuffer(func(buf *bytes.Buffer) {
				if s.currentPointerName != "" {
					s.printfStyle("ref", "%v ", s.currentPointerName)
					s.currentPointerName = ""
				}

				s.printf("(")
				s.printfStyle("ref", "0x%08x", value.Pointer())
				s.printf(")")
			}))

			s.printf("&")
			s.dumpVal(value.Elem())
			for _, comment := range previousComments {
				s.AddComment(comment)
			}
		}

	case reflect.Struct:
		s.DumpStructType(typ)
		s.printf("{%s\n", s.DumpStructComments(value))
		s.DepthDown()
		s.DumpStructFields(value, nil)
		s.DepthUp()
		s.Pad()
		s.printf("}")

	case reflect.Array, reflect.Slice:
		n := value.Len()

		var w io.Writer
		buf := &bytes.Buffer{}
		w, s.w = s.w, buf

		if kind == reflect.Array {
			s.printfStyle("meta", "[%v]", n)
		} else {
			s.printf("[]")
		}
		s.printfStyle("meta", "%v", typ.Elem())
		s.w = w

		if kind == reflect.Slice && value.IsNil() {
			s.AddComment(buf.String())

			s.printfStyle("ref", "nil")
		} else {
			s.printf("%s{", buf.String())

			if len(s.comments) > 0 && n/ElementsPerLine > 1 || s.forceNewLines {
				s.print(s.formatComments())
				s.ResetComments()
			}

			if kind == reflect.Slice {
				s.AddComment(fmt.Sprintf("len=%d", n))
			}

			s.DepthDown()
			for i := 0; i < n; i++ {
				if s.breakLineIfNecessary(n, i) {
					s.printf(" ")
				}
				s.dumpVal(value.Index(i))
				s.printf(",")
			}
			s.DepthUp()

			s.breakLineIfNecessary(n, 0)
			s.printf("}")
		}

		if s.depth == 0 {
			s.printf(s.DumpStructComments(value))
		}

	case reflect.Map:
		str := s.WithTempBuffer(func(buf *bytes.Buffer) {
			s.printfStyle("note", "map")
			s.printf("[")
			s.printfStyle("meta", "%s", typ.Key().Name())
			s.printf("]")
			s.printfStyle("meta", "%v", typ.Elem())
		})

		if value.IsNil() {
			s.AddComment(str)

			s.printfStyle("ref", "nil")
		} else {
			s.printf("%s{", str)

			keys := value.MapKeys()
			sort.Sort(mapKeysSorter{
				keys: keys,
			})
			n := len(keys)

			s.DepthDown()
			for i, k := range keys {
				if s.breakLineIfNecessary(n, i) {
					s.printf(" ")
				}

				s.dumpVal(k)
				s.printf(": ")
				s.dumpVal(value.MapIndex(k))

				s.printf(",")
			}
			s.DepthUp()

			s.breakLineIfNecessary(n, 0)
			s.printf("}")
		}

		if s.depth == 0 {
			s.printf(s.DumpStructComments(value))
		}

	case reflect.Chan:
		str := s.WithTempBuffer(func(buf *bytes.Buffer) {
			switch typ.ChanDir() {
			case reflect.RecvDir:
				s.printfStyle("note", "<-chan")
			case reflect.SendDir:
				s.printfStyle("note", "chan<-")
			default:
				s.printfStyle("note", "chan")
			}
			s.printfStyle("meta", " %v", value.Type().Elem())
		})

		if value.IsNil() {
			s.AddComment(str)
			s.printfStyle("ref", "nil")
		} else {
			if capacity := value.Cap(); capacity > 0 {
				s.printf("make(%s, %v)", str, capacity)
			} else {
				s.printf("make(%s)", str)
			}
		}

		if s.depth == 0 {
			s.printf(s.DumpStructComments(value))
		}

	case reflect.Func:
		if !value.IsValid() {
			return
		}

		str := s.WithTempBuffer(func(buf *bytes.Buffer) {
			s.printfStyle("meta", "%v", value.Type())
		})

		if value.IsNil() {
			s.AddComment(str)
			s.printfStyle("ref", "nil")
		} else {
			s.printf(str)
		}

		if s.depth == 0 {
			s.printf(s.DumpStructComments(value))
		}

	case reflect.Interface:
		if value.IsNil() {
			s.printfStyle("ref", "nil")
		} else {
			previousForceDumpTypeInstantiation := s.forceDumpTypeInstantiation
			s.forceDumpTypeInstantiation = true
			// Let's go take the value inside the non-nil interface.
			// Goes deeper inside "generic" structures, maps, arrays or slices.
			s.dumpVal(value.Elem())
			s.forceDumpTypeInstantiation = previousForceDumpTypeInstantiation
		}

	default:
		if value.CanInterface() {
			s.printf("%v", value.Interface())
		} else {
			s.printf("%v", value.String())
		}
	}
}
