// Package policy holds tests that enforce the scanner's privacy promises.
// It has no code of its own.
//
// The scanner makes no network connections. The tests check this in two ways:
//
//  1. No Go file in this repository may import a network package (net,
//     net/*, crypto/tls or golang.org/x/net). There are no exceptions.
//  2. Of every package compiled into the Windows scanner, packages that can
//     make web or secure connections (net/http, crypto/tls, net/rpc,
//     net/smtp, golang.org/x/net) must not appear at all.
//
// The one unavoidable exception: golang.org/x/sys/windows, the Go team's
// standard Windows library that the design allows, imports the low-level
// "net" package for type definitions used by its IP helper functions. The
// scanner never calls those functions. The test allows "net" to appear only
// because of that library, and fails if anything else brings it in.
//
// Opening the report page at the end is done by asking Windows to open an
// address in the user's own browser (rundll32 url.dll), which is not a
// network connection made by the scanner.
//
// The Android app (android/) is held to the same promise. Its manifest may
// ask only for QUERY_ALL_PACKAGES (to list the apps), so Android itself
// refuses any connection, and no Java file may import network or web-view
// packages. CI also checks the permissions of the built app.
package policy
