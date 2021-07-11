package main

import (
	"bytes"
	"fmt"
	"syscall/js"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func decryptPDF(input []uint8) ([]byte, error) {
	r := bytes.NewReader(input)
	var w bytes.Buffer

	pdfcpu.ConfigPath = "disable"
	conf := pdfcpu.NewDefaultConfiguration()
	conf.Cmd = pdfcpu.DECRYPT

	if err := api.Optimize(r, &w, conf); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}

func main() {
	fmt.Println("Hello from Go WebAssembly!")
	js.Global().Set("decryptPDF", pdfWrapper())
	<-make(chan bool)
}

func pdfWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) != 1 {
			return "Invalid no of arguments passed"
		}

		pdfBuf := args[0]

		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resolve := args[0]
			reject := args[1]

			go func() {
				inBuf := make([]uint8, pdfBuf.Get("byteLength").Int())

				js.CopyBytesToGo(inBuf, pdfBuf)

				output, err := decryptPDF(inBuf)
				if err != nil {
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())
					reject.Invoke(errorObject)
				} else {
					dst := js.Global().Get("Uint8Array").New(len(output))
					js.CopyBytesToJS(dst, output)
					resolve.Invoke(dst)
				}
			}()

			return nil
		})

		promiseConstructor := js.Global().Get("Promise")
		return promiseConstructor.New(handler)
	})
}
