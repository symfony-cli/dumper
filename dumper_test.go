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
	"fmt"
	"image"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unsafe"

	"bufio"
	"net/http"

	. "gopkg.in/check.v1"
)

type DumperSuite struct{}

var _ = Suite(&DumperSuite{})

func TestDumper(t *testing.T) { TestingT(t) }

type dumpEqualsChecker struct {
	*CheckerInfo
}

var DumpEquals Checker = &dumpEqualsChecker{
	&CheckerInfo{Name: "DumpEquals", Params: []string{"obtained", "expected"}},
}

var memoryRegexp = regexp.MustCompile("0x[0-9a-f]{8,12}")

func (checker *dumpEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	defer func() {
		if v := recover(); v != nil {
			result = false
			error = fmt.Sprint(v)
		}
	}()
	obtained := params[0].(string)
	obtained = memoryRegexp.ReplaceAllString(obtained, "0xXXXXXXXXXX")
	obtained = strings.TrimRight(obtained, "\n")
	return obtained == params[1], ""
}

func (ts *DumperSuite) TestString(c *C) {
	c.Assert(Sdump("foo"), DumpEquals, "\"foo\"")
}

func (ts *DumperSuite) TestBool(c *C) {
	c.Assert(Sdump(true), DumpEquals, "true")

	c.Assert(Sdump(false), DumpEquals, "false")
}

func (ts *DumperSuite) TestNumbers(c *C) {
	cases := map[string]interface{}{
		// integers
		"5":             5,
		"-5":            -5,
		"int8(42)":      int8(42),
		"int16(-42)":    int16(-42),
		"int32(5)":      int32(5),
		"int32(-5)":     int32(-5),
		"int64(987659)": int64(987659),
		// unsigned integers
		"uint(5)":        uint(5),
		"uint8(42)":      uint8(42),
		"uint16(42)":     uint16(42),
		"uint32(5)":      uint32(5),
		"uint64(987659)": uint64(987659),
		// floats
		"37.2":        37.2,
		"float32(37)": float32(37),
		// complexes
		"3":                        complex128(3),
		"complex(3, 1)":            complex(3, 1),
		"complex64(complex(3, 2))": complex64(complex(3, 2)),
	}

	for expected, value := range cases {
		c.Assert(Sdump(value), DumpEquals, expected)
	}
}

func (ts *DumperSuite) TestBasic(c *C) {
	type Person struct {
		Name string
		Age  int
	}

	s := Sdump(Person{
		Name: "Bob",
		Age:  20,
	})

	c.Assert(s, DumpEquals, `dumper.Person{
  Name: "Bob",
  Age: 20,
}`)
}

func (ts *DumperSuite) TestBasic2(c *C) {
	type Person struct {
		Name   string
		Age    int
		Parent *Person
	}

	s := Sdump(Person{
		Name: "Bob",
		Age:  20,
		Parent: &Person{
			Name: "Jane",
			Age:  50,
		},
	})

	c.Assert(s, DumpEquals, `dumper.Person{
  Name: "Bob",
  Age: 20,
  Parent: &dumper.Person{ // (0xXXXXXXXXXX)
    Name: "Jane",
    Age: 50,
    Parent: nil, // &dumper.Person
  },
}`)
}

func (ts *DumperSuite) TestStruct(c *C) {
	c.Assert(Sdump(DumpEquals), DumpEquals, `&dumper.dumpEqualsChecker{ // (0xXXXXXXXXXX)
  CheckerInfo: &check.v1.CheckerInfo{ // (0xXXXXXXXXXX)
    Name: "DumpEquals",
    Params: []string{"obtained", "expected",}, // len=2
  },
}`)

	val := image.Point{X: 1, Y: 2}
	c.Assert(Sdump(val), DumpEquals, `image.Point{
  X: 1,
  Y: 2,
}`)
}

func (ts *DumperSuite) TestAnonymousStruct(c *C) {
	foo := struct {
		Foo string
		Bar interface{}
	}{
		Foo: "hello",
	}

	c.Assert(Sdump(foo), DumpEquals, `struct { Foo string; Bar interface {} }{ // anonymous struct
  Foo: "hello",
  Bar: nil,
}`)
}

func (ts *DumperSuite) TestArray(c *C) {
	// Types should disappear within the arrays
	intArray := [2]int{}
	c.Check(Sdump(intArray), DumpEquals, `[2]int{0, 0,}`)

	floatArray := [2]float64{}
	c.Check(Sdump(floatArray), DumpEquals, `[2]float64{0, 0,}`)

	floatArray = [2]float64{1, 2}
	c.Check(Sdump(floatArray), DumpEquals, `[2]float64{1, 2,}`)

	complexArray := [2]complex128{1, complex(0, 1)}
	c.Check(Sdump(complexArray), DumpEquals, `[2]complex128{1, complex(0, 1),}`)

	var foo *[2]byte
	c.Check(Sdump(foo), DumpEquals, `nil // &[2]uint8`)

	foo = &[2]byte{}
	c.Check(Sdump(foo), DumpEquals, `&[2]uint8{0, 0,} // (0xXXXXXXXXXX)`)
}

func (ts *DumperSuite) TestLongArray(c *C) {
	var foo *[90]byte
	c.Assert(Sdump(foo), DumpEquals, `nil // &[90]uint8`)

	foo = &[90]byte{}
	c.Assert(Sdump(foo), DumpEquals, `&[90]uint8{ // (0xXXXXXXXXXX)
  0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
}`)
}

func (ts *DumperSuite) TestSlice(c *C) {
	var foo []byte
	c.Check(Sdump(foo), DumpEquals, `nil // []uint8`)

	foo = []byte{'a', 'b'}
	c.Check(Sdump(foo), DumpEquals, `[]uint8{97, 98,} // len=2`)
}

func (ts *DumperSuite) TestMap(c *C) {
	var foo map[string]bool
	c.Assert(Sdump(foo), DumpEquals, `nil // map[string]bool`)

	foo = make(map[string]bool)
	c.Assert(Sdump(foo), DumpEquals, `map[string]bool{}`)

	foo["foo"] = true
	c.Assert(Sdump(foo), DumpEquals, `map[string]bool{"foo": true,}`)
}

func (ts *DumperSuite) TestInterfaces(c *C) {
	type Foo struct {
		Bar interface{}
	}

	foo := Foo{
		Bar: 5,
	}
	c.Assert(Sdump(foo), DumpEquals, `dumper.Foo{
  Bar: 5,
}`)

	foo = Foo{
		Bar: "hello",
	}
	c.Assert(Sdump(foo), DumpEquals, `dumper.Foo{
  Bar: "hello",
}`)

	foo = Foo{
		Bar: uint(5),
	}
	c.Assert(Sdump(foo), DumpEquals, `dumper.Foo{
  Bar: uint(5),
}`)
}

func (ts *DumperSuite) TestCircularRef(c *C) {
	type Circular struct {
		Foo string
		Bar *Circular
	}

	foo := &Circular{
		Foo: "hello",
	}
	foo.Bar = foo

	c.Assert(Sdump(foo), DumpEquals, `&dumper.Circular{ // p0 (0xXXXXXXXXXX)
  Foo: "hello",
  Bar: p0,
}`)
}

func (ts *DumperSuite) TestPointer(c *C) {
	type a struct {
		Foo string
	}
	foo := a{
		Foo: "hello",
	}

	c.Assert(Sdump(reflect.ValueOf(&foo).Pointer()), DumpEquals, `uintptr(0xXXXXXXXXXX)`)
	var bar uintptr
	bar = reflect.ValueOf(&foo).Pointer()
	bar = uintptr(0x0101010101)
	c.Assert(Sdump(bar), DumpEquals, `uintptr(0xXXXXXXXXXX)`)

	ptr := unsafe.Pointer(reflect.ValueOf(&foo).Pointer())
	ptr = unsafe.Pointer(uintptr(0x0101010101))
	c.Assert(Sdump(ptr), DumpEquals, `unsafe.Pointer(uintptr(0xXXXXXXXXXX))`)
}

func (ts *DumperSuite) TestChan(c *C) {
	var receiveOnly chan<- int
	c.Assert(Sdump(receiveOnly), DumpEquals, `nil // chan<- int`)

	receiveOnly = make(chan<- int)
	c.Assert(Sdump(receiveOnly), DumpEquals, `make(chan<- int)`)

	ch := make(chan bool)
	c.Assert(Sdump(ch), DumpEquals, `make(chan bool)`)

	bufferedCh := make(chan bool, 1)
	c.Assert(Sdump(bufferedCh), DumpEquals, `make(chan bool, 1)`)
}

func (ts *DumperSuite) TestFunc(c *C) {
	var foo func() string
	c.Assert(Sdump(foo), DumpEquals, `nil // func() string`)

	c.Assert(Sdump(func() string { return "foo" }), DumpEquals, `func() string`)
}

func (ts *DumperSuite) TestCustomType(c *C) {
	type testCustomType func() string
	var f testCustomType

	f = func() string { return "foo" }

	c.Assert(Sdump(f), DumpEquals, `dumper.testCustomType`)
}

func (ts *DumperSuite) TestCustomDumperExternal(c *C) {
	reader := bufio.NewReader(strings.NewReader(`HTTP/1.1 200 OK
Content-Type: text/html; charset=UTF-8
Server: nginx/1.4.6 (Ubuntu)
Vary: Accept-Encoding
Vary: Authorization

Hello World!
`))
	resp, err := http.ReadResponse(reader, nil)
	c.Assert(err, IsNil)

	// c.Assert(Sdump(resp), DumpEquals, `&http.Response{
	c.Assert(Sdump(resp), DumpEquals, `&http.Response{ // (0xXXXXXXXXXX)
  Status: "200 OK",
  StatusCode: 200,
  Proto: "HTTP/1.1",
  TransferEncoding: nil, // []string
  ContentLength: -1, // int64
  Headers: {
    "Content-Type": "text/html; charset=UTF-8",
    "Server": "nginx/1.4.6 (Ubuntu)",
    "Vary": "Accept-Encoding",
    "Vary": "Authorization",
  },
  Body: "Hello World!
",
}`)
}

func (ts *DumperSuite) TestCustomDumper(c *C) {
	var foo interface{}
	foo = TestCustomDumper{X: 42, Y: 43, Z: 44}

	c.Assert(Sdump(foo), DumpEquals, `dumper.TestCustomDumper{ // Custom comment
  X: 44,
  Y: 45,
  // you can really display whatever you want
  Z: -5,
}`)
}

type TestCustomDumper struct {
	X, Y, Z int
}

func (t TestCustomDumper) Dump(s State) {
	fmt.Fprintf(s, `  X: %v,
  Y: %v,
  // you can really display whatever you want
`, t.Z, t.Z+1)
	s.DumpStructField("Z", reflect.ValueOf(-5))

	s.AddComment("Custom comment")
}

func (ts *DumperSuite) TestPrivateFields(c *C) {
	type Person struct {
		Name string
		Age  int
		// this one should be shown
		private string
	}

	s := Sdump(Person{
		Name:    "Bob",
		Age:     20,
		private: "foo",
	})

	c.Assert(s, DumpEquals, `dumper.Person{
  Name: "Bob",
  Age: 20,
  private: "foo",
}`)

	UnregisterCustomDumper(http.Request{})
	c.Assert(Sdump(http.Request{}), DumpEquals, httpRequestExceptedDump)

	RegisterCustomDumper(http.Request{}, DumpStructWithPrivateFields)
	c.Assert(Sdump(http.Request{}), DumpEquals, httpRequestExceptedDumpWithPrivateFields)
}
