// Package paths implements absolute, component-boundary-aware scene paths.
//
// A scene path looks like "/Show/OldNamespace/Materials/Red". Migration is a
// re-rooting of a namespace subtree: "/Show/OldNamespace" -> "/Show/Shot/Set".
// Rewriting must happen on whole path components so that a look-alike sibling
// such as "/Show/OldNamespaceArchive" is never treated as a descendant.
package paths

import "strings"

// Separator is the path component separator.
const Separator = "/"

// Clean normalizes an absolute slash path: duplicate slashes are collapsed and
// a trailing slash (other than the root) is removed.
func Clean(p string) string {
	if p == "" {
		return ""
	}
	absolute := strings.HasPrefix(p, Separator)
	parts := strings.Split(p, Separator)
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		kept = append(kept, part)
	}
	out := strings.Join(kept, Separator)
	if absolute {
		out = Separator + out
	}
	if out == "" && absolute {
		out = Separator
	}
	return out
}

// Components returns the non-empty path components.
func Components(p string) []string {
	clean := Clean(p)
	if clean == "" || clean == Separator {
		return nil
	}
	return strings.Split(strings.TrimPrefix(clean, Separator), Separator)
}

// Join joins elements into a cleaned absolute path.
func Join(elem ...string) string {
	return Clean(strings.Join(elem, Separator))
}

// Base returns the final path component ("/" returns "/").
func Base(p string) string {
	clean := Clean(p)
	if clean == Separator {
		return Separator
	}
	idx := strings.LastIndex(clean, Separator)
	if idx < 0 {
		return clean
	}
	return clean[idx+1:]
}

// Parent returns the parent path. The root reports ok=false.
func Parent(p string) (parent string, ok bool) {
	clean := Clean(p)
	if clean == Separator || clean == "" {
		return Separator, false
	}
	idx := strings.LastIndex(clean, Separator)
	if idx == 0 {
		return Separator, true
	}
	if idx < 0 {
		return ".", false
	}
	return clean[:idx], true
}

// HasPrefix reports whether p equals prefix or is a descendant of prefix on
// component boundaries. Consequently HasPrefix("/OldNamespace2",
// "/OldNamespace") is false even though the raw strings share a prefix.
func HasPrefix(p, prefix string) bool {
	cleanP := Clean(p)
	cleanPrefix := Clean(prefix)
	if cleanPrefix == Separator {
		return strings.HasPrefix(cleanP, Separator)
	}
	return cleanP == cleanPrefix || strings.HasPrefix(cleanP, cleanPrefix+Separator)
}

// RewritePrefix re-roots p when it is oldRoot or lies beneath oldRoot,
// replacing only that component prefix with newRoot. It returns the rewritten
// path and true when a rewrite occurred. Paths outside oldRoot are returned
// unchanged with rewritten=false.
func RewritePrefix(oldRoot, newRoot, p string) (string, bool) {
	oldRoot = Clean(oldRoot)
	newRoot = Clean(newRoot)
	cleanP := Clean(p)
	if !HasPrefix(cleanP, oldRoot) {
		return p, false
	}
	if cleanP == oldRoot {
		return newRoot, true
	}
	rel := strings.TrimPrefix(cleanP, oldRoot+Separator)
	return Join(newRoot, rel), true
}

// RawStringReplace performs the unsafe operation the feedback warns about:
// a literal strings.ReplaceAll of the old namespace text. It is provided so
// tests and diagnostics can contrast it with RewritePrefix. It both misses the
// parent-child re-root bookkeeping and corrupts component-boundary look-alikes.
func RawStringReplace(oldText, newText, p string) string {
	return strings.ReplaceAll(p, oldText, newText)
}
