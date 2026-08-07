//go:build js && wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"
)

// Web saves live in browser localStorage under the "qsim.save." prefix.
// Downloading is a browser file download; uploading uses a hidden file input;
// sharing talks to the same-origin share API (see share-service/).

const (
	saveKeyPrefix = "qsim.save."
)

func jsLocalStorage() js.Value {
	return js.Global().Get("localStorage")
}

func platformListSaves() []string {
	ls := jsLocalStorage()
	var names []string
	for i := 0; i < ls.Get("length").Int(); i++ {
		key := ls.Call("key", i).String()
		if strings.HasPrefix(key, saveKeyPrefix) {
			names = append(names, strings.TrimPrefix(key, saveKeyPrefix))
		}
	}
	return names
}

func platformReadSave(name string) (string, error) {
	val := jsLocalStorage().Call("getItem", saveKeyPrefix+name)
	if val.Type() == js.TypeNull {
		return "", errNotFound
	}
	return val.String(), nil
}

func platformWriteSave(name string, data string) error {
	jsLocalStorage().Call("setItem", saveKeyPrefix+name, data)
	return nil
}

func platformRemoveSave(name string) error {
	jsLocalStorage().Call("removeItem", saveKeyPrefix+name)
	return nil
}

func platformDownloadSave(name string, data string) {
	doc := js.Global().Get("document")
	blob := js.Global().Get("Blob").New([]any{data}, map[string]any{"type": "text/plain"})
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	a := doc.Call("createElement", "a")
	a.Set("href", url)
	a.Set("download", name)
	doc.Get("body").Call("appendChild", a)
	a.Call("click")
	a.Call("remove")
	js.Global().Get("URL").Call("revokeObjectURL", url)
}

// platformRequestSaveUpload wires up a temporary <input type="file"> element
// and calls cb with the picked file's name and contents.
func platformRequestSaveUpload(cb func(name string, data string)) {
	doc := js.Global().Get("document")
	input := doc.Call("createElement", "input")
	input.Set("type", "file")
	input.Set("accept", ".qsim,text/plain,application/json")

	var onchange js.Func
	onchange = js.FuncOf(func(this js.Value, args []js.Value) any {
		onchange.Release()
		ev := args[0]
		files := ev.Get("target").Get("files")
		if files.Get("length").Int() == 0 {
			return nil
		}
		file := files.Index(0)
		name := file.Get("name").String()
		reader := js.Global().Get("FileReader").New()
		var onload js.Func
		onload = js.FuncOf(func(this js.Value, args []js.Value) any {
			onload.Release()
			cb(name, reader.Get("result").String())
			return nil
		})
		reader.Set("onload", onload)
		reader.Call("readAsText", file)
		return nil
	})
	input.Call("addEventListener", "change", onchange)
	input.Call("click")
}

// jsResult carries the outcome of a JS promise.
type jsResult struct {
	value js.Value
	err   error
}

func jsValueString(v js.Value) string {
	if v.Type() == js.TypeObject {
		if msg := v.Get("message"); !msg.IsUndefined() && msg.Type() == js.TypeString {
			return msg.String()
		}
	}
	return v.String()
}

// awaitPromise blocks the calling goroutine until the JS promise settles and
// returns its resolution value or rejection error.
func awaitPromise(p js.Value) (js.Value, error) {
	ch := make(chan jsResult, 1)
	then := js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- jsResult{value: args[0]}
		return nil
	})
	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- jsResult{err: fmt.Errorf("%s", jsValueString(args[0]))}
		return nil
	})
	p.Call("then", then, catch)
	r := <-ch
	then.Release()
	catch.Release()
	return r.value, r.err
}

// fetchJSON performs a fetch and returns the parsed JSON object.
func fetchJSON(path string, method string, body string) (js.Value, error) {
	init := map[string]any{"method": method}
	if body != "" {
		init["body"] = body
		init["headers"] = map[string]any{"Content-Type": "application/json"}
	}
	resp, err := awaitPromise(js.Global().Call("fetch", path, init))
	if err != nil {
		return js.Value{}, err
	}
	if !resp.Get("ok").Bool() {
		return js.Value{}, fmt.Errorf("share service error: %s", resp.Get("statusText").String())
	}
	return awaitPromise(resp.Call("json"))
}

func platformShareUpload(data string) (string, error) {
	obj, err := fetchJSON(sharePath, "POST", fmt.Sprintf(`{"data":%q}`, data))
	if err != nil {
		return "", err
	}
	id := obj.Get("id").String()
	if id == "" {
		return "", fmt.Errorf("share service returned no id")
	}
	base := platformShareBaseURL()
	return base + "/?share=" + id, nil
}

func platformShareFetch(id string) (string, error) {
	obj, err := fetchJSON(sharePath+"/"+id, "GET", "")
	if err != nil {
		return "", err
	}
	data := obj.Get("data").String()
	if data == "" {
		return "", errNotFound
	}
	return data, nil
}

func platformShareBaseURL() string {
	return js.Global().Get("location").Get("origin").String()
}

// platformCopyToClipboard uses the browser clipboard.writeText API.
func platformCopyToClipboard(text string) {
	nav := js.Global().Get("navigator")
	if nav.IsUndefined() || nav.Get("clipboard").IsUndefined() {
		return
	}
	p := nav.Get("clipboard").Call("writeText", text)
	awaitPromise(p)
}

// shareIDFromURL extracts the ?share=<id> parameter, if any, from the current
// page URL. Called once at startup to auto-load a shared circuit.
func shareIDFromURL() string {
	params := js.Global().Get("location").Get("search").String()
	if !strings.HasPrefix(params, "?") {
		return ""
	}
	for _, pair := range strings.Split(strings.TrimPrefix(params, "?"), "&") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 && kv[0] == "share" {
			return kv[1]
		}
	}
	return ""
}