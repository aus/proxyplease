package proxyplease

import "fmt"

var debugf = func(format string, a ...interface{}) {
	if a == nil {
		fmt.Println(format)
	} else {
		fmt.Println(fmt.Printf(format, a))
	}

}

// SetDebugf sets a debugf function for debug output
func SetDebugf(f func(format string, a ...interface{})) {
	debugf = f
}
