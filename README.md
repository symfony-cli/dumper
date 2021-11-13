Go Dumper
=========

When debugging Go code, being able to dump values with nice formatting and
colors can go a long way:

```go
package main

import (
    "time"

    . "github.com/symfony-cli/dumper"
)

func main() {
    now := time.Now().Local()
    Dump(now)
}
```

Output:

```
time.Time{{
  wall: 507838786, // uint64
  ext: 63649250679, // int64
  loc: &time.Location { // (0x0014c3a0)
    name: "",
    zone: nil, // []time.zone
    tx: nil, // []time.zoneTrans
    cacheStart: 0, // int64,
    cacheEnd: 0, // int64,
    cacheZone: nil, // &time.zone
  },
}
```

Custom Dumpers
--------------

Implement the `Dumpable` interface on your types to take control of how your
type is dumped:

``` go
type Dumpable interface {
    Dump(State)
}
```

Use the available helpers on the `State` object to write your custom dump.
Alternatively, you can write directly to the provided stream. In both cases,
your type name (and eventually the pointer comment) will be automatically
added.

Read he test suite for some examples.

Another alternative, useful for package you don't maintain is to register a
function of type `DumpFunc`:

```go
func init() {
    RegisterCustomDumper(http.Request{}, dumpHttpRequest)
}

func dumpHttpRequest(s State, v reflect.Value) {
    for _, f := range []string{"Status", "Proto"} {
        s.DumpStructField(f, v.FieldByName(f))
    }
    s.AddComment("Hello World!")
}
```
