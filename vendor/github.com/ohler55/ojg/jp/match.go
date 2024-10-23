// Copyright (c) 2024, Peter Ohler, All rights reserved.

package jp

// PathMatch returns true if the provided path would match the target
// expression. The path argument is expected to be a normalized path with only
// elements of Root ($), At (@), Child (string), or Nth (int). A Filter
// fragment in the target expression will match any value in path since it
// requires data from a JSON document to be evaluated. Slice fragments always
// return true as long as the path element is an Nth.
func PathMatch(target, path Expr) bool {
	if 0 < len(target) {
		switch target[0].(type) {
		case Root, At:
			target = target[1:]
		}
	}
	if 0 < len(path) {
		switch path[0].(type) {
		case Root, At:
			path = path[1:]
		}
	}
	for i, f := range target {
		if len(path) == 0 {
			return false
		}
		switch path[0].(type) {
		case Child, Nth:
		default:
			return false
		}
		switch tf := f.(type) {
		case Child, Nth:
			if tf != path[0] {
				return false
			}
			path = path[1:]
		case Bracket:
			// ignore and don't advance path
		case Wildcard:
			path = path[1:]
		case Union:
			var ok bool
			for _, u := range tf {
			check:
				switch tu := u.(type) {
				case string:
					if Child(tu) == path[0] {
						ok = true
						break check
					}
				case int64:
					if Nth(tu) == path[0] {
						ok = true
						break check
					}
				}
			}
			if !ok {
				return false
			}
			path = path[1:]
		case Slice:
			if _, ok := path[0].(Nth); !ok {
				return false
			}
			path = path[1:]
		case *Filter:
			// Assume a match since there is no data for comparison.
			path = path[1:]
		case Descent:
			rest := target[i+1:]
			for 0 < len(path) {
				if PathMatch(rest, path) {
					return true
				}
				path = path[1:]
			}
			return false
		default:
			return false
		}
	}
	return true
}
