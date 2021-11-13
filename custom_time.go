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
