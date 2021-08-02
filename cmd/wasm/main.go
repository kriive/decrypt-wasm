// cmd/wasm/main.go
package main

import (
	"bytes"
	"fmt"
	"syscall/js"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// DecryptPDF decrypts a PDF and returns the unencrypted file.
// It uses pdfcpu, it's a bit of a hack, because I don't know
// about API stability of the package and this could break in a update.
// Who knows? \U0001f937
func DecryptPDF(input []byte) ([]byte, error) {
	r := bytes.NewReader(input)
	var w bytes.Buffer

	// Workaround: avoid pdfcpu to complain for missing config
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

		// Outer function has one argument, which is the input locked PDF file
		pdfBuf := args[0]

		handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// This handler function will be wrapped inside a Promise, see below
			// The first argument is a function called if everything's ok,
			// the second one is a function called when something's gone wrong
			resolve := args[0]
			reject := args[1]

			// Execute the decrypt function in a separate goroutine
			go func() {
				inBuf := make([]byte, pdfBuf.Get("byteLength").Int())

				// Copy the pdfBuf (containing the locked PDF as an array)
				// to the Go WebAssembly runtime
				js.CopyBytesToGo(inBuf, pdfBuf)

				// Do our little magic trick and call our decryption function
				output, err := DecryptPDF(inBuf)
				if err != nil {
					// Create a new Error wrapping the error message
					// returned by DecryptPDF
					errorConstructor := js.Global().Get("Error")
					errorObject := errorConstructor.New(err.Error())

					// Invoke the sad path, calling the reject function
					reject.Invoke(errorObject)
				} else {
					// Create a new Uint8Array with the proper length
					dst := js.Global().Get("Uint8Array").New(len(output))

					// Copy the decrypted output from Go runtime to JS
					js.CopyBytesToJS(dst, output)

					// We're done, call the happy path
					resolve.Invoke(dst)
				}
			}()

			return nil
		})

		// Create the Promise
		promiseConstructor := js.Global().Get("Promise")

		// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise
		// A new Promise can be created from a function with two parameters
		// In our case: resolve (happy path) and reject (sad path, an error occurred)
		return promiseConstructor.New(handler)
	})
}
