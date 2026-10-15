package utils

// MaxLogLine is the longest single line of build output Builder captures.
// bufio.Scanner's default (64 KB) is too small for minified bundles and
// long compiler diagnostics.
const MaxLogLine = 16 * 1024 * 1024
