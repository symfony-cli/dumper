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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
)

type State interface {
	io.Writer
	WithTempBuffer(func(buf *bytes.Buffer)) string

	AddComment(string)
	ResetComments() []string

	ForceNewLines(bool) bool

	DepthUp()
	DepthDown()

	Pad()
	Dump(value interface{})
	DumpString(str string)
	DumpScalar(v interface{}, t reflect.Type, dumpTypeInstantiation bool)
	DumpComplex(v complex128, t reflect.Kind)
	DumpStructType(t reflect.Type)
	DumpStructField(string, reflect.Value)
}

var (
	dumpableType  = reflect.TypeOf((*Dumpable)(nil)).Elem()
	customDumpers = make(map[reflect.Type]DumpFunc)
)

// Dumpable is the interface for implementing custom dumper for your types.
type Dumpable interface {
	Dump(State)
}

type DumpFunc func(State, reflect.Value)

type dumpableFn struct {
	v  reflect.Value
	fn DumpFunc
}

func (d *dumpableFn) Dump(s State) {
	d.fn(s, d.v)
}

func UnregisterCustomDumper(v interface{}) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Interface {
		if val.IsNil() {
			return
		}

		val = val.Elem()
	}

	delete(customDumpers, val.Type())
}

func RegisterCustomDumper(v interface{}, f DumpFunc) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Interface {
		if val.IsNil() {
			return
		}

		val = val.Elem()
	}

	customDumpers[val.Type()] = f
}

func (s *state) dumpCustomFn(v reflect.Value, fn DumpFunc) {
	d := &dumpableFn{v: v, fn: fn}

	s.dumpCustom(v, d)
}

func (s *state) dumpCustom(v reflect.Value, vv Dumpable) {
	previousComments := s.ResetComments()
	s.DepthDown()

	str := s.WithTempBuffer(func(buf *bytes.Buffer) {
		scanner := bufio.NewScanner(strings.NewReader(s.WithTempBuffer(func(buf *bytes.Buffer) {
			vv.Dump(s)
		})))

		for scanner.Scan() {
			line := strings.TrimRight(scanner.Text(), " \n\t")

			if len(line) > 0 {
				s.print(line)
			}

			s.Write([]byte("\n"))
		}

		if err := scanner.Err(); err != nil {
			s.AddComment(fmt.Sprintf("Invalid input: %s", err))
		}
	})

	for _, comment := range previousComments {
		s.AddComment(comment)
	}

	s.DumpStructType(v.Type())
	s.printf("{%s\n%s", s.DumpStructComments(v), str)
	s.DepthUp()
	s.Pad()
	s.printf("}")
}
