package main

import (
	"fmt"
	_ "unsafe"
)

//go:cgo_import_dynamic libc_getpid getpid "libc.so.6"
//go:cgo_import_dynamic _ _ "libc.so.6"

func main() {
	fmt.Println("dynamic import ok")
}
