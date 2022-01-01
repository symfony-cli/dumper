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
	"reflect"
	"sort"
)

type visitedPointer struct {
	idx     int
	visited bool
}

type visitedPointersMap map[uintptr]visitedPointer

type pointerMap struct {
	pointers       []uintptr
	reusedPointers visitedPointersMap
}

func mapPointers(v reflect.Value) visitedPointersMap {
	pm := &pointerMap{
		reusedPointers: make(visitedPointersMap),
	}
	pm.consider(v)
	return pm.reusedPointers
}

// Recursively consider v and each of its children, updating the map according to the
// semantics of MapReusedPointers
func (pm *pointerMap) consider(v reflect.Value) {
	if v.Kind() == reflect.Invalid {
		return
	}

	if isPointerValue(v) && v.Pointer() != 0 { // pointer is 0 for unexported fields
		reused := pm.addPointerReturnTrueIfWasReused(v.Pointer())
		if reused {
			// No use descending inside this value, since it have been seen before and all its descendants
			// have been considered
			return
		}
	}

	// Now descend into any children of this value
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		numEntries := v.Len()
		for i := 0; i < numEntries; i++ {
			pm.consider(v.Index(i))
		}

	case reflect.Interface:
		pm.consider(v.Elem())

	case reflect.Ptr:
		pm.consider(v.Elem())

	case reflect.Map:
		keys := v.MapKeys()
		sort.Sort(mapKeysSorter{
			keys: keys,
		})
		for _, key := range keys {
			pm.consider(v.MapIndex(key))
		}

	case reflect.Struct:
		numFields := v.NumField()
		for i := 0; i < numFields; i++ {
			pm.consider(v.Field(i))
		}
	}
}

// addPointer to the pointerMap, update reusedPointers. Returns true if pointer was reused
func (pm *pointerMap) addPointerReturnTrueIfWasReused(ptr uintptr) bool {
	// Is this already known to be reused?
	if _, have := pm.reusedPointers[ptr]; have {
		return true
	}

	// Have we seen it once before?
	for _, seen := range pm.pointers {
		if ptr == seen {
			// Add it to the register of pointers we have seen more than once
			pm.reusedPointers[ptr] = visitedPointer{idx: len(pm.reusedPointers)}
			return true
		}
	}

	// This pointer was new to us
	pm.pointers = append(pm.pointers, ptr)
	return false
}

func (s *state) handleCircularRef(value reflect.Value) bool {
	pointerName, alreadyVisited := s.pointerRef(value)
	if pointerName == "" {
		return false
	}
	if !alreadyVisited {
		s.currentPointerName = pointerName
		return false
	}

	s.printfStyle("ref", pointerName)
	return true
}

// registers that the value has been visited and checks to see if it is one of the
// pointers we will see multiple times. If it is, it returns a temporary name for this
// pointer. It also returns a boolean value indicating whether this is the first time
// this name is returned so the caller can decide whether the contents of the pointer
// has been dumped before or not.
func (s *state) pointerRef(v reflect.Value) (string, bool) {
	if isPointerValue(v) {
		ptr := v.Pointer()

		if ptrV, isReused := s.pointers[ptr]; isReused {
			alreadyVisited := ptrV.visited
			if !alreadyVisited {
				s.pointers[ptr] = visitedPointer{idx: ptrV.idx, visited: true}
			}
			return fmt.Sprintf("p%d", ptrV.idx), alreadyVisited
		}
	}

	return "", false
}

func isPointerValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Slice, reflect.Ptr, reflect.UnsafePointer:
		return true
	}

	return false
}
