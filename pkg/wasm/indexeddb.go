//go:build js && wasm

package wasm

import (
	"encoding/base64"
	"fmt"
	"syscall/js"
)

const (
	idbName  = "bytewire"
	idbStore = "offline"
	idbKey   = "queue"
)

// persistQueue serializes GlobalQueue frames to a JSON array of base64 strings
// and stores them in IndexedDB. Fire-and-forget async.
func persistQueue() {
	frames := GlobalQueue.Flush()
	if len(frames) == 0 {
		return
	}

	// Re-enqueue so they aren't lost
	for _, f := range frames {
		GlobalQueue.Enqueue(f)
	}

	// Build JS array of base64 strings
	arr := js.Global().Get("Array").New()
	for _, f := range frames {
		arr.Call("push", base64.StdEncoding.EncodeToString(f))
	}

	jsonStr := js.Global().Get("JSON").Call("stringify", arr).String()
	writeToIDB(jsonStr)
}

// loadPersistedQueue loads frames from IndexedDB and enqueues them into GlobalQueue.
// This blocks until the IndexedDB read completes.
func loadPersistedQueue() {
	done := make(chan string, 1)

	openReq := js.Global().Get("indexedDB").Call("open", idbName, 1)
	openReq.Set("onupgradeneeded", js.FuncOf(func(_ js.Value, args []js.Value) any {
		db := args[0].Get("target").Get("result")
		if !db.Call("objectStoreNames").Call("contains", idbStore).Bool() {
			db.Call("createObjectStore", idbStore)
		}
		return nil
	}))
	openReq.Set("onsuccess", js.FuncOf(func(_ js.Value, args []js.Value) any {
		db := args[0].Get("target").Get("result")
		tx := db.Call("transaction", idbStore, "readonly")
		store := tx.Call("objectStore", idbStore)
		getReq := store.Call("get", idbKey)
		getReq.Set("onsuccess", js.FuncOf(func(_ js.Value, args []js.Value) any {
			result := args[0].Get("target").Get("result")
			if result.IsUndefined() || result.IsNull() {
				done <- ""
				return nil
			}
			done <- result.String()
			return nil
		}))
		getReq.Set("onerror", js.FuncOf(func(_ js.Value, _ []js.Value) any {
			done <- ""
			return nil
		}))
		return nil
	}))
	openReq.Set("onerror", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		done <- ""
		return nil
	}))

	jsonStr := <-done
	if jsonStr == "" {
		return
	}

	// Parse JSON array of base64 strings
	parsed := js.Global().Get("JSON").Call("parse", jsonStr)
	length := parsed.Get("length").Int()
	for i := 0; i < length; i++ {
		b64 := parsed.Index(i).String()
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			fmt.Println("bytewire: failed to decode persisted frame:", err)
			continue
		}
		GlobalQueue.Enqueue(data)
	}
	fmt.Printf("bytewire: loaded %d persisted offline frames\n", length)
}

// clearPersistedQueue removes the stored queue from IndexedDB. Fire-and-forget.
func clearPersistedQueue() {
	openReq := js.Global().Get("indexedDB").Call("open", idbName, 1)
	openReq.Set("onupgradeneeded", js.FuncOf(func(_ js.Value, args []js.Value) any {
		db := args[0].Get("target").Get("result")
		if !db.Call("objectStoreNames").Call("contains", idbStore).Bool() {
			db.Call("createObjectStore", idbStore)
		}
		return nil
	}))
	openReq.Set("onsuccess", js.FuncOf(func(_ js.Value, args []js.Value) any {
		db := args[0].Get("target").Get("result")
		tx := db.Call("transaction", idbStore, "readwrite")
		store := tx.Call("objectStore", idbStore)
		store.Call("delete", idbKey)
		return nil
	}))
}

// writeToIDB stores a JSON string in IndexedDB. Fire-and-forget.
func writeToIDB(jsonStr string) {
	openReq := js.Global().Get("indexedDB").Call("open", idbName, 1)
	openReq.Set("onupgradeneeded", js.FuncOf(func(_ js.Value, args []js.Value) any {
		db := args[0].Get("target").Get("result")
		if !db.Call("objectStoreNames").Call("contains", idbStore).Bool() {
			db.Call("createObjectStore", idbStore)
		}
		return nil
	}))
	openReq.Set("onsuccess", js.FuncOf(func(_ js.Value, args []js.Value) any {
		db := args[0].Get("target").Get("result")
		tx := db.Call("transaction", idbStore, "readwrite")
		store := tx.Call("objectStore", idbStore)
		store.Call("put", jsonStr, idbKey)
		return nil
	}))
}
