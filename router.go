// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

// Package httprouter is a trie based high performance HTTP request router.
//
// A trivial example is:
//
//  package main
//
//  import (
//      "fmt"
//      "github.com/julienschmidt/httprouter"
//      "net/http"
//      "log"
//  )
//
//  func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
//      fmt.Fprint(w, "Welcome!\n")
//  }
//
//  func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
//      fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
//  }
//
//  func main() {
//      router := httprouter.New()
//      router.GET("/", Index)
//      router.GET("/hello/:name", Hello)
//
//      log.Fatal(http.ListenAndServe(":8080", router))
//  }
//
// The router matches incoming requests by the request method and the path.
// If a handle is registered for this path and method, the router delegates the
// request to that function.
// For the methods GET, POST, PUT, PATCH and DELETE shortcut functions exist to
// register handles, for all other methods router.HandleMethod can be used.
//
// The registered path, against which the router matches incoming requests, can
// contain two types of parameters:
//  Syntax    Type
//  :name     named parameter
//  *name     catch-all parameter
//
// Named parameters are dynamic path segments. They match anything until the
// next '/' or the path end:
//  Path: /blog/:category/:post
//
//  Requests:
//   /blog/go/request-routers            match: category="go", post="request-routers"
//   /blog/go/request-routers/           no match, but the router would redirect
//   /blog/go/                           no match
//   /blog/go/request-routers/comments   no match
//
// Catch-all parameters match anything until the path end, including the
// directory index (the '/' before the catch-all). Since they match anything
// until the end, catch-all parameters must always be the final path element.
//  Path: /files/*filepath
//
//  Requests:
//   /files/                             match: filepath="/"
//   /files/LICENSE                      match: filepath="/LICENSE"
//   /files/templates/article.html       match: filepath="/templates/article.html"
//   /files                              no match, but the router would redirect
//
// The value of parameters is saved as a slice of the Param struct, consisting
// each of a key and a value. The slice is passed to the HandleMethod func as a third
// parameter.
// There are two ways to retrieve the value of a parameter:
//  // by the name of the parameter
//  user := ps.ByName("user") // defined by :user or *user
//
//  // by the index of the parameter. This way you can also get the name (key)
//  thirdKey   := ps[2].Key   // the name of the 3rd parameter
//  thirdValue := ps[2].Value // the value of the 3rd parameter
package httprouter

import (
	"context"
	"net/http"
)

// any are all the methods that are handled
var any = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// Params are the path params resolved from the path
type Params map[string]string

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) string {
	if val, ok := ps[name]; ok {
		return val
	}
	return ""
}

// Router is a http.Handler which can be used to dispatch requests to different
// handler functions via configurable routes
type Router struct {
	trees map[string]*node

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 307 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 307 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool

	// If enabled, the router automatically replies to OPTIONS requests.
	// Custom OPTIONS handlers take priority over automatic replies.
	HandleOPTIONS bool

	// Configurable http.Handler which is called when no matching route is
	// found. If it is not set, http.NotFound is used.
	NotFound http.Handler

	// Configurable http.Handler which is called when a request
	// cannot be routed and HandleMethodNotAllowed is true.
	// If it is not set, http.Error with http.StatusMethodNotAllowed is used.
	// The "Allow" header with allowed request methods is set before the handler
	// is called.
	MethodNotAllowed http.Handler
}

// Make sure the Router conforms with the http.Handler interface
var _ http.Handler = New()

// New returns a new initialized Router.
// Path auto-correction, including trailing slashes, is enabled by default.
func New() *Router {
	return &Router{
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
	}
}

// Get is a shortcut for router.HandleMethod("GET", path, handler)
func (r *Router) Get(path string, handler http.Handler) {
	r.HandleMethod("GET", path, handler)
}

// Head is a shortcut for router.HandleMethod("HEAD", path, handler)
func (r *Router) Head(path string, handler http.Handler) {
	r.HandleMethod("HEAD", path, handler)
}

// Options is a shortcut for router.HandleMethod("OPTIONS", path, handler)
func (r *Router) Options(path string, handler http.Handler) {
	r.HandleMethod("OPTIONS", path, handler)
}

// Post is a shortcut for router.HandleMethod("POST", path, handler)
func (r *Router) Post(path string, handler http.Handler) {
	r.HandleMethod("POST", path, handler)
}

// Put is a shortcut for router.HandleMethod("PUT", path, handler)
func (r *Router) Put(path string, handler http.Handler) {
	r.HandleMethod("PUT", path, handler)
}

// Patch is a shortcut for router.HandleMethod("PATCH", path, handler)
func (r *Router) Patch(path string, handler http.Handler) {
	r.HandleMethod("PATCH", path, handler)
}

// Delete is a shortcut for router.HandleMethod("DELETE", path, handler)
func (r *Router) Delete(path string, handler http.Handler) {
	r.HandleMethod("DELETE", path, handler)
}

// HandleMethod registers a new request handler with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) HandleMethod(method, path string, handler http.Handler) {
	r.HandleMethodFunc(method, path, handler.ServeHTTP)
}

func (r *Router) HandleMethods(methods []string, path string, handler http.Handler) {
	r.HandleMethodsFunc(methods, path, handler.ServeHTTP)
}

func (r *Router) Handle(path string, handler http.Handler) {
	r.HandleMethodsFunc(any, path, handler.ServeHTTP)
}

// Get is a shortcut for router.HandleMethodFunc("GET", path, handleFunc)
func (r *Router) GetFunc(path string, handleFunc http.HandlerFunc) {
	r.HandleMethodFunc("GET", path, handleFunc)
}

// Head is a shortcut for router.HandleMethodFunc("HEAD", path, handleFunc)
func (r *Router) HeadFunc(path string, handleFunc http.HandlerFunc) {
	r.HandleMethodFunc("HEAD", path, handleFunc)
}

// Options is a shortcut for router.HandleMethodFunc("OPTIONS", path, handleFunc)
func (r *Router) OptionsFunc(path string, handleFunc http.HandlerFunc) {
	r.HandleMethodFunc("OPTIONS", path, handleFunc)
}

// Post is a shortcut for router.HandleMethodFunc("POST", path, handleFunc)
func (r *Router) PostFunc(path string, handleFunc http.HandlerFunc) {
	r.HandleMethodFunc("POST", path, handleFunc)
}

// Put is a shortcut for router.HandleMethodFunc("PUT", path, handleFunc)
func (r *Router) PutFunc(path string, handleFunc http.HandlerFunc) {
	r.HandleMethodFunc("PUT", path, handleFunc)
}

// Patch is a shortcut for router.HandleMethodFunc("PATCH", path, handleFunc)
func (r *Router) PatchFunc(path string, handleFunc http.HandlerFunc) {
	r.HandleMethodFunc("PATCH", path, handleFunc)
}

// Delete is a shortcut for router.HandleMethodFunc("DELETE", path, handleFunc)
func (r *Router) DeleteFunc(path string, handleFunc http.HandlerFunc) {
	r.HandleMethodFunc("DELETE", path, handleFunc)
}

// HandleMethod registers a new request handle function with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) HandleMethodFunc(method, path string, handleFunc http.HandlerFunc) {
	if path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}

	if r.trees == nil {
		r.trees = make(map[string]*node)
	}

	root := r.trees[method]
	if root == nil {
		root = new(node)
		r.trees[method] = root
	}
	root.addRoute(path, handleFunc)
}

func (r *Router) HandleMethodsFunc(methods []string, path string, handleFunc http.HandlerFunc) {
	for _, method := range methods {
		r.HandleMethodFunc(method, path, handleFunc)
	}
}

// ServeFiles serves files from the given file system root.
// The path must end with "/*filepath", files are then served from the local
// path /defined/root/dir/*filepath.
// For example if root is "/etc" and *filepath is "passwd", the local file
// "/etc/passwd" would be served.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// To use the operating system's file system implementation,
// use http.Dir:
//     router.ServeFiles("/src/*filepath", http.Dir("/var/www"))
func (r *Router) ServeFiles(path string, root http.FileSystem) {
	if len(path) < 10 || path[len(path)-10:] != "/*filepath" {
		panic("path must end with /*filepath in path '" + path + "'")
	}

	fileServer := http.FileServer(root)
	r.Get(path, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		req.URL.Path = ContextParams(req.Context())["filepath"]
		fileServer.ServeHTTP(w, req)
	}))
}

func (r *Router) allowed(path, reqMethod string) (allow string) {
	if path == "*" { // server-wide
		for method := range r.trees {
			if method == "OPTIONS" {
				continue
			}

			// add request method to list of allowed methods
			if len(allow) == 0 {
				allow = method
			} else {
				allow += ", " + method
			}
		}
	} else { // specific path
		for method := range r.trees {
			// Skip the requested method - we already tried this one
			if method == reqMethod || method == "OPTIONS" {
				continue
			}

			handle, _, _ := r.trees[method].getValue(path)
			if handle != nil {
				// add request method to list of allowed methods
				if len(allow) == 0 {
					allow = method
				} else {
					allow += ", " + method
				}
			}
		}
	}
	if len(allow) > 0 {
		allow += ", OPTIONS"
	}
	return
}

// ServeHTTP makes the router implement the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path

	if root := r.trees[req.Method]; root != nil {
		if handler, ps, tsr := root.getValue(path); handler != nil {
			if ps != nil {
				// Merge if there are already params in the context
				// Only the non existing params from the previous context will be merged
				if p, ok := req.Context().Value(ParamsContextKey).(Params); ok {
					for k, v := range p {
						if _, ok := ps[k]; !ok {
							ps[k] = v
						}
					}
				}
				req = req.WithContext(context.WithValue(req.Context(), ParamsContextKey, ps))
			}
			handler(w, req)
			return
		} else if req.Method != "CONNECT" && path != "/" {
			code := 301 // Permanent redirect, request with GET method
			if req.Method != "GET" {
				// Temporary redirect, request with same method
				// As of Go 1.3, Go does not support status code 308.
				code = 307
			}

			if tsr && r.RedirectTrailingSlash {
				if len(path) > 1 && path[len(path)-1] == '/' {
					req.URL.Path = path[:len(path)-1]
				} else {
					req.URL.Path = path + "/"
				}
				http.Redirect(w, req, req.URL.String(), code)
				return
			}

			// Try to fix the request path
			if r.RedirectFixedPath {
				fixedPath, found := root.findCaseInsensitivePath(
					CleanPath(path),
					r.RedirectTrailingSlash,
				)
				if found {
					req.URL.Path = string(fixedPath)
					http.Redirect(w, req, req.URL.String(), code)
					return
				}
			}
		}
	}

	if req.Method == "OPTIONS" {
		// HandleMethod OPTIONS requests
		if r.HandleOPTIONS {
			if allow := r.allowed(path, req.Method); len(allow) > 0 {
				w.Header().Set("Allow", allow)
				return
			}
		}
	} else {
		// HandleMethod 405
		if r.HandleMethodNotAllowed {
			if allow := r.allowed(path, req.Method); len(allow) > 0 {
				w.Header().Set("Allow", allow)
				if r.MethodNotAllowed != nil {
					r.MethodNotAllowed.ServeHTTP(w, req)
				} else {
					http.Error(w,
						http.StatusText(http.StatusMethodNotAllowed),
						http.StatusMethodNotAllowed,
					)
				}
				return
			}
		}
	}

	// HandleMethod 404
	if r.NotFound != nil {
		r.NotFound.ServeHTTP(w, req)
	} else {
		http.NotFound(w, req)
	}
}
