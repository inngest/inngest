// Package types names the string identifiers that Inngest uses, such as app
// and function slugs.  it holds identifier types only and imports nothing
// from this module, so any package can import it without an import cycle.
package types

// AppSlug is an app's ID, as the client sets it.  Connect sends it as the app
// name in the handshake.
type AppSlug string

// FnSlug is a function's fully qualified ID, in the form "<app>-<function>".
type FnSlug string
