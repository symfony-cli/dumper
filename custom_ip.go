package dumper

import (
	"net"
	"reflect"
)

func init() {
	RegisterCustomDumper(net.IP{}, dumpNetIp)
}

func dumpNetIp(s State, v reflect.Value) {
	ip := v.Interface().(net.IP)
	s.DumpString(ip.String())
}
