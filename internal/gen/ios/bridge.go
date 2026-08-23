package ios

// iosBridgeSource is overlaid into the app main package so c-archive
// exports EletrocromoStart without editing the app tree.
const iosBridgeSource = `//go:build ios

package main

import "C"

import "os"

//export EletrocromoStart
func EletrocromoStart(readyFile *C.char) {
	os.Setenv("ELETROCROMO_NO_UI", "1")
	os.Setenv("ELETROCROMO_NO_ENSURE", "1")
	os.Setenv("NO_PROXY", "127.0.0.1,localhost,::1")
	os.Setenv("no_proxy", "127.0.0.1,localhost,::1")
	if readyFile != nil {
		os.Setenv("ELETROCROMO_READY_FILE", C.GoString(readyFile))
	}
	main()
}
`
