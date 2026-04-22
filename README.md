# Issue Tracker

A Cloudflare Workers application built with Go (Syumai Workers) that fetches recent GitHub issues from repositories stored in GistDB and send message via WhatsApp

## Technical Stuff

### Why syscall/js?

Go's standard `http.Client` doesn't work in Cloudflare Workers WASM environment due to incorrect `this` binding when calling JavaScript's `fetch` API. This project uses `syscall/js` to directly call the fetch API with proper context binding.

### GistDB Integration

The application uses GistDB to store the list of repositories to track. This allows dynamic repository management without redeploying the worker.

## License

MIT
