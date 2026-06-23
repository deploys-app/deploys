package auth

import "net/http"

// setLoopbackHeaders applies the safety headers shared by the success and error
// pages: no referrer (so the code never leaks via a Referer header) and no
// caching. Neither page reflects any request data, so there is no XSS surface in
// the 127.0.0.1 origin that can read the code.
func setLoopbackHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
}

const successHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>deploys.app</title></head>
<body style="font-family:system-ui,sans-serif;text-align:center;margin-top:6rem">
<h1>Login complete</h1>
<p>You are signed in to the deploys.app CLI. You can close this tab.</p>
</body></html>`

const errorHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>deploys.app</title></head>
<body style="font-family:system-ui,sans-serif;text-align:center;margin-top:6rem">
<h1>Login failed</h1>
<p>Return to your terminal and run <code>deploys login</code> again.</p>
</body></html>`

func writeSuccessPage(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(successHTML))
	flush(w)
}

func writeErrorPage(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(errorHTML))
	flush(w)
}

// flush pushes the buffered response onto the socket before the caller signals
// completion. Without it, the listener can be closed (RFC 8252 §7.3: shut down
// the instant the code arrives) before the response reaches the browser, which
// then shows a connection reset instead of the page.
func flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
