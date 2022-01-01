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
	"time"
)

func init() {
	RegisterCustomDumper(time.Time{}, dumpTime)
}

func dumpTime(s State, v reflect.Value) {
	t := v.Interface().(time.Time)
	s.AddComment(fmt.Sprintf("@%v", t.Unix()))
	s.DumpStructField("date", reflect.ValueOf(t.Format("2006-01-02 15:04:05.999999 MST (Z07:00)")))
}
