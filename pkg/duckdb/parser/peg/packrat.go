package peg

import (
	"fmt"
	"strings"
	"sync"
)

type NodeKind int

const (
	KindRule NodeKind = iota
	KindLiteral
	KindPrimitive
	KindSeq
	KindChoice
	KindOptional
	KindRepeat
	KindNot
)

type Node struct {
	Kind     NodeKind
	Name     string // KindRule/KindPrimitive: the rule/primitive name.
	Text     string // KindLiteral/KindPrimitive: raw matched source text.
	Value    string // KindPrimitive only, when HasValue: a decoded value the primitive sets.
	HasValue bool   // Value was explicitly set — distinguishes "decoded to an empty string" from "no decoded value at all".
	// Value used to be `any`, letting a primitive decode to anything. In
	// practice every primitive in this codebase (the only place this
	// engine's used) always decodes to a string, and Value is on the
	// single hottest allocation path there is — a primitive is tried
	// speculatively very often during backtracking — so boxing a string
	// into an interface on every match was a real, measured cost for no
	// actual benefit. If a future caller genuinely needs a non-string
	// decoded value, widen this back to `any` then.
	Alt        int // KindChoice only: index of the matched alternative.
	Start, End int
	Children   []*Node
}

// Primitive matches one leaf token type starting at pos, which is always
// past any trivia the Parser already skipped. It returns ok=false without
// side effects if it doesn't match here. On a match, it must build its
// result via newNode(...) rather than &Node{...} — that's what lets a
// matched primitive live in the same bump-allocated slab as every other
// node in the tree instead of its own heap allocation (primitives are tried
// speculatively very often during backtracking, so this matters a lot).
type Primitive func(newNode func(Node) *Node, input string, pos int) (node *Node, next int, ok bool)

// Parser packrat-parses input against Grammar. It does not support left
// recursion (direct or mutual) — a rule whose first alternative starts by
// referencing itself, directly or through another rule, recurses until the
// call stack overflows rather than terminating. This is a deliberate scope
// boundary, not an oversight: DuckDB's own grammar never needs left
// recursion (every precedence chain is written as "Head TailRule*", not
// "TailRule* Head"), so this engine doesn't implement Warth-style
// seed-growing to support it.
type Parser struct {
	Grammar    *Grammar
	Primitives map[string]Primitive
	// SkipTrivia advances past whitespace/comments between tokens. Defaults
	// to skipping nothing if nil — this package has no notion of "SQL";
	// the root package (Task 4) supplies a real SQL trivia-skipper.
	SkipTrivia func(input string, pos int) int

	dispatchOnce sync.Once
	dispatch     map[string]dispatchEntry

	// sessionPool recycles evaluators (and their node/child/env slabs, memo
	// maps, and skip cache) across Session.Parse calls — see Session's doc
	// comment. Parser.Parse never touches this: it always builds a fresh
	// evaluator, so its returned *Node is valid indefinitely.
	sessionPool sync.Pool
}

// dispatchEntry merges Primitives and Grammar.Rules into one lookup table,
// built once per Parser (it's a pure function of Grammar/Primitives, both
// fixed for the Parser's lifetime) rather than probed as two separate maps
// on every evalRef call — a bare rule reference is the single hottest
// operation in a parse, so cutting its lookup from two map probes (try
// Primitives, fall back to Grammar.Rules) to one matters. Primitives take
// priority on a name collision, matching the old two-map lookup order.
//
// id is a per-Parser-instance interned index for this name, assigned once
// here. evalRef/evalCall already have the dispatchEntry in hand by the time
// they build a memo key, so using id there instead of the name string
// itself is free (no extra lookup) and shrinks refMemoKey/callMemoKey by a
// 16-byte string header — every memoized rule invocation in the parse
// carries one of these keys, so the saving is real.
type dispatchEntry struct {
	id   int
	prim Primitive // set if this name is a primitive.
	rule *Rule     // set otherwise, if this name is a grammar rule.
}

func (p *Parser) getDispatch() map[string]dispatchEntry {
	p.dispatchOnce.Do(func() {
		d := make(map[string]dispatchEntry, len(p.Primitives)+len(p.Grammar.Rules))
		nextID := 0
		for name, prim := range p.Primitives {
			d[name] = dispatchEntry{id: nextID, prim: prim}
			nextID++
		}
		for name, rule := range p.Grammar.Rules {
			if _, exists := d[name]; exists {
				continue
			}
			d[name] = dispatchEntry{id: nextID, rule: rule}
			nextID++
		}
		p.dispatch = d
	})
	return p.dispatch
}

type ParseError struct {
	Pos     int
	Message string
}

func (e *ParseError) Error() string { return fmt.Sprintf("offset %d: %s", e.Pos, e.Message) }

// Parse builds a fresh evaluator for this one call, so the returned *Node
// is valid indefinitely. A caller that parses many inputs back-to-back and
// fully consumes each tree before moving to the next (e.g. immediately
// adapting it into some other owned representation, as ParseString does)
// should use AcquireSession instead — see Session's doc comment for why.
func (p *Parser) Parse(input string, root string) (*Node, error) {
	e := p.newEvaluator(input)
	return e.run(root)
}

// newEvaluator allocates a fresh evaluator sized for input. The capacity
// guesses below are all proportional to input size and calibrated against
// measured usage on representative queries — refMemo entries and
// nodeSlab/childSlab elements run 7-20x the input length (more for small
// inputs, where fixed grammar overhead isn't yet amortized; the multipliers
// undershoot small inputs a bit and target the large end, since that's
// where a mid-parse regrowth — a full map rehash, or copying an abandoned
// slab — actually costs something). envSlab (parameterized-rule call
// frames) and callMemo (evalCall's memo, same call frames plus repeat
// lookups) both run much lower, around 0.3-0.6x, since a grammar has only a
// couple of parameterized rules. The goal is avoiding that regrowth, not
// eliminating all headroom.
func (p *Parser) newEvaluator(input string) *evaluator {
	skip := p.SkipTrivia
	if skip == nil {
		skip = func(input string, pos int) int { return pos }
	}
	skipCache := make([]int, len(input)+1)
	for i := range skipCache {
		skipCache[i] = -1
	}
	e := &evaluator{
		dispatch: p.getDispatch(), skip: skip, input: input,
		refMemo:   make(map[refMemoKey]memoEntry, len(input)*12),
		callMemo:  make(map[callMemoKey]memoEntry, len(input)/2+1),
		skipCache: skipCache,
		nodeSlab:  make([]Node, 0, len(input)*8),
		childSlab: make([]*Node, 0, len(input)*8),
		envSlab:   make([]env, 0, len(input)/2+1),
	}
	e.newNodeFn = e.newNode
	return e
}

// reset reuses e's slabs, memo maps, and skip cache for a new input,
// instead of the fresh allocations newEvaluator would make. clear(m) empties
// a map without discarding its backing table (no rehash/realloc), and
// truncating a slice to length 0 keeps its backing array — both keep
// whatever capacity a previous, similarly-sized input already grew them to.
// dispatch/skip never change for a given Parser, so they're left alone.
func (e *evaluator) reset(input string) {
	e.input = input
	e.furthestFail = 0
	if cap(e.skipCache) >= len(input)+1 {
		e.skipCache = e.skipCache[:len(input)+1]
	} else {
		e.skipCache = make([]int, len(input)+1)
	}
	for i := range e.skipCache {
		e.skipCache[i] = -1
	}
	e.nodeSlab = e.nodeSlab[:0]
	e.childSlab = e.childSlab[:0]
	e.envSlab = e.envSlab[:0]
	clear(e.refMemo)
	clear(e.callMemo)
}

// run parses e.input (already set by newEvaluator or reset) against root.
func (e *evaluator) run(root string) (*Node, error) {
	start := e.skipPos(0)
	node, next, ok := e.evalRef(root, nil, false, start)
	if !ok {
		return nil, &ParseError{Pos: e.furthestFail, Message: fmt.Sprintf("failed to parse %s", root)}
	}
	end := e.skipPos(next)
	if end != len(e.input) {
		return nil, &ParseError{Pos: e.furthestFail, Message: "unexpected trailing input"}
	}
	return node, nil
}

// Session lets a caller that parses many inputs back-to-back reuse one
// evaluator's node/child/env slabs, memo maps, and skip cache across calls
// instead of paying for fresh ones on every Parse — see newEvaluator's doc
// comment for what those buffers are. This is NOT a drop-in replacement for
// Parser.Parse: the *Node tree returned by Session.Parse is only valid
// until the next call to Session.Parse or Session.Release on the same
// Session, since reusing the buffers means exactly that memory gets
// overwritten. A caller must fully consume the tree — e.g. by walking it
// into some other owned representation — before calling either. Not safe
// for concurrent use; acquire one Session per goroutine. Call Release
// exactly once, and don't use the Session again afterward — Release puts
// it back in the pool for some other, unrelated caller to pick up.
type Session struct {
	parser *Parser
	e      *evaluator
}

// AcquireSession returns a Session, reusing one from a prior Release if the
// pool has one available.
func (p *Parser) AcquireSession() *Session {
	if v := p.sessionPool.Get(); v != nil {
		return v.(*Session)
	}
	return &Session{parser: p}
}

// Release returns the session's buffers to the pool for a future
// AcquireSession call to reuse. The caller must be done reading any *Node
// this session returned before calling this — see the Session doc comment.
func (s *Session) Release() {
	s.parser.sessionPool.Put(s)
}

func (s *Session) Parse(input, root string) (*Node, error) {
	if s.e == nil {
		s.e = s.parser.newEvaluator(input)
	} else {
		s.e.reset(input)
	}
	return s.e.run(root)
}

type binding struct {
	expr Expr
	env  *env
}

type env struct {
	name   string
	bind   binding
	parent *env
}

func (e *env) lookup(name string) (binding, bool) {
	for cur := e; cur != nil; cur = cur.parent {
		if cur.name == name {
			return cur.bind, true
		}
	}
	return binding{}, false
}

// refMemoKey memoizes evalRef — a bare, unparameterized rule reference,
// which is by far the more common of the two memoized call shapes (a
// typical grammar has only a couple of parameterized rules, e.g. List/
// Parens, versus every other rule reference in the grammar going through
// this path). Keeping it separate from callMemoKey means the dominant
// share of memo entries doesn't carry the argKind/sig fields it never uses.
type refMemoKey struct {
	ruleID int // dispatchEntry.id, not the rule name — see dispatchEntry's doc comment.
	pos    int
	noSkip bool
}

// callMemoKey memoizes evalCall — a parameterized-rule invocation like
// `List(Expression)` — where argKind/sig additionally identify which
// argument expression the rule's parameter was bound to.
type callMemoKey struct {
	ruleID  int // dispatchEntry.id, not the rule name — see dispatchEntry's doc comment.
	argKind ExprKind
	sig     string
	pos     int
	noSkip  bool
}

type memoEntry struct {
	node *Node
	next int
	ok   bool
}

type evaluator struct {
	dispatch     map[string]dispatchEntry
	skip         func(input string, pos int) int
	input        string
	refMemo      map[refMemoKey]memoEntry
	callMemo     map[callMemoKey]memoEntry
	furthestFail int

	// skipCache memoizes skipPos by input offset (-1 = uncomputed). Every
	// layer of a pure rule-reference chain re-skips trivia at the same
	// already-skipped position (see evalRef's comment on why that's
	// idempotent), so this turns that redundant rescanning into an O(1)
	// lookup after the first time each position is visited.
	skipCache []int

	// nodeSlab/childSlab bump-allocate *Node values and their Children
	// backing arrays out of a shared, ever-growing slice instead of one
	// heap allocation per node/slice. A pointer or sub-slice handed out
	// today stays valid forever: append only ever extends past the
	// current length, never rewrites an index already handed out, so nothing
	// already-returned (including memoized results) is ever aliased by a
	// later write, even across backtracking that abandons a partial match.
	nodeSlab  []Node
	childSlab []*Node

	// envSlab bump-allocates the *env frames evalCall builds for each
	// parameterized-rule invocation, same rationale as nodeSlab/childSlab.
	envSlab []env

	// newNodeFn is e.newNode as a func value, computed once instead of at
	// every primitive call site. A method value like e.newNode allocates a
	// fresh closure each time it's evaluated (it escapes here since it's
	// passed through an indirect call the compiler can't see into) — with
	// primitives tried speculatively as often as they are, re-evaluating
	// it per call would cost as much as the heap allocation it replaces.
	newNodeFn func(Node) *Node
}

func (e *evaluator) fail(pos int) {
	if pos > e.furthestFail {
		e.furthestFail = pos
	}
}

// skipPos is a memoized wrapper around e.skip: SkipTrivia is a pure function
// of (input, pos), and this package re-derives the same already-skipped
// position repeatedly (see evalRef's comment), so caching by offset avoids
// redoing that scan.
func (e *evaluator) skipPos(pos int) int {
	if pos < len(e.skipCache) {
		if v := e.skipCache[pos]; v >= 0 {
			return v
		}
		v := e.skip(e.input, pos)
		e.skipCache[pos] = v
		return v
	}
	return e.skip(e.input, pos)
}

// newNode hands out a *Node backed by a shared, amortized-growth slab
// instead of its own heap allocation.
func (e *evaluator) newNode(n Node) *Node {
	e.nodeSlab = append(e.nodeSlab, n)
	return &e.nodeSlab[len(e.nodeSlab)-1]
}

// newEnv is newNode's counterpart for the *env frames evalCall builds.
func (e *evaluator) newEnv(v env) *env {
	e.envSlab = append(e.envSlab, v)
	return &e.envSlab[len(e.envSlab)-1]
}

// oneChild wraps a single node in a fresh length-1, cap-1 slice carved from
// the shared childSlab (cap is capped at len so an errant append elsewhere
// reallocates instead of clobbering whatever the slab appends next). Safe to
// call unconditionally: n is already fully computed, so this is one atomic
// append with no recursive call landing in the middle of it.
func (e *evaluator) oneChild(n *Node) []*Node {
	e.childSlab = append(e.childSlab, n)
	i := len(e.childSlab)
	return e.childSlab[i-1 : i : i]
}

// newChildren copies a fully-computed slice of children into the shared
// childSlab in one shot, returning a cap-limited view over the copy. Callers
// must finish all recursive evaluation first (see the ExprSeq comment) —
// this must be the only thing appending to childSlab for the duration of the
// copy.
func (e *evaluator) newChildren(nodes []*Node) []*Node {
	start := len(e.childSlab)
	e.childSlab = append(e.childSlab, nodes...)
	end := len(e.childSlab)
	return e.childSlab[start:end:end]
}

// argSig identifies a parameterized-rule argument for the memo key, paired
// with the returned ExprKind so callers don't need a string prefix to tell
// two kinds apart (e.g. a rule named "Other" as an ExprRef vs. the
// catch-all default below). The ExprRef/ExprLiteral cases — overwhelmingly
// the common ones, since this grammar only ever binds List/Parens'
// parameter to a bare rule reference — return the existing field directly
// with no allocation; only the rare nested-ExprCall argument falls back to
// building a string.
func argSig(expr Expr) (ExprKind, string) {
	switch expr.Kind {
	case ExprRef:
		return ExprRef, expr.Ref
	case ExprCall:
		var b strings.Builder
		b.WriteString(expr.Ref)
		b.WriteByte('(')
		for i, c := range expr.Children {
			if i > 0 {
				b.WriteByte(',')
			}
			_, s := argSig(c)
			b.WriteString(s)
		}
		b.WriteByte(')')
		return ExprCall, b.String()
	case ExprLiteral:
		return ExprLiteral, expr.Literal
	default:
		return expr.Kind, ""
	}
}

// evalRef resolves a bare rule/parameter reference: parameter bindings in
// env take priority (transparent substitution — the result is returned
// as-is, with no extra wrapper node), then Primitives, then the rule table.
func (e *evaluator) evalRef(name string, env *env, noSkip bool, pos int) (*Node, int, bool) {
	if b, ok := env.lookup(name); ok {
		return e.evalExpr(b.expr, b.env, noSkip, pos)
	}
	d, ok := e.dispatch[name]
	if !ok {
		panic(fmt.Sprintf("peg: rule %q not found", name))
	}
	if d.prim != nil {
		p := pos
		if !noSkip {
			p = e.skipPos(p)
		}
		node, next, ok := d.prim(e.newNodeFn, e.input, p)
		if !ok {
			e.fail(p)
			return nil, 0, false
		}
		node.Name = name
		return node, next, true
	}
	rule := d.rule
	if rule.Param != "" {
		panic(fmt.Sprintf("peg: rule %q is parameterized and needs an argument", name))
	}
	// Skip trivia before recording Start, not just before matching the
	// eventual leaf token — otherwise a rule reached only through several
	// layers of pure rule-references (no literal/primitive of its own)
	// gets a Start that includes whatever whitespace preceded it, since
	// only leaf matching (matchLiteral, a Primitive) skips trivia
	// otherwise. Idempotent to call again here even when a nested leaf
	// will also skip: skipping from an already-non-trivia position is a
	// no-op.
	p := pos
	if !noSkip {
		p = e.skipPos(p)
	}
	key := refMemoKey{ruleID: d.id, pos: p, noSkip: noSkip}
	if m, ok := e.refMemo[key]; ok {
		return m.node, m.next, m.ok
	}
	body, next, ok := e.evalExpr(rule.Expr, nil, noSkip, p)
	var result *Node
	if ok {
		result = e.newNode(Node{Kind: KindRule, Name: name, Start: p, End: next, Children: e.oneChild(body)})
	}
	e.refMemo[key] = memoEntry{node: result, next: next, ok: ok}
	return result, next, ok
}

// evalCall resolves a parameterized-rule invocation (`Name(Arg)`): binds
// the callee's single parameter to Arg *closed over the caller's env*, then
// evaluates the callee's body in a fresh environment containing only that
// one binding.
func (e *evaluator) evalCall(name string, arg Expr, callerEnv *env, noSkip bool, pos int) (*Node, int, bool) {
	d, ok := e.dispatch[name]
	rule := d.rule
	if !ok || rule == nil || rule.Param == "" {
		panic(fmt.Sprintf("peg: %q is not a parameterized rule", name))
	}
	// See evalRef's matching comment: skip trivia before recording Start.
	p := pos
	if !noSkip {
		p = e.skipPos(p)
	}
	argKind, argSigStr := argSig(arg)
	key := callMemoKey{ruleID: d.id, argKind: argKind, sig: argSigStr, pos: p, noSkip: noSkip}
	if m, ok := e.callMemo[key]; ok {
		return m.node, m.next, m.ok
	}
	calleeEnv := e.newEnv(env{name: rule.Param, bind: binding{expr: arg, env: callerEnv}})
	body, next, ok := e.evalExpr(rule.Expr, calleeEnv, noSkip, p)
	var result *Node
	if ok {
		result = e.newNode(Node{Kind: KindRule, Name: name, Start: p, End: next, Children: e.oneChild(body)})
	}
	e.callMemo[key] = memoEntry{node: result, next: next, ok: ok}
	return result, next, ok
}

func (e *evaluator) evalExpr(expr Expr, env *env, noSkip bool, pos int) (*Node, int, bool) {
	switch expr.Kind {
	case ExprLiteral:
		next, ok := e.matchLiteral(expr.Literal, noSkip, pos)
		if !ok {
			return nil, 0, false
		}
		return e.newNode(Node{Kind: KindLiteral, Text: expr.Literal, Start: pos, End: next}), next, true
	case ExprClass:
		p := pos
		if !noSkip {
			p = e.skipPos(p)
		}
		if p >= len(e.input) {
			e.fail(p)
			return nil, 0, false
		}
		r := rune(e.input[p])
		if !expr.Class.Matches(r) {
			e.fail(p)
			return nil, 0, false
		}
		return e.newNode(Node{Kind: KindLiteral, Text: e.input[p : p+1], Start: p, End: p + 1}), p + 1, true
	case ExprRef:
		return e.evalRef(expr.Ref, env, noSkip, pos)
	case ExprCall:
		return e.evalCall(expr.Ref, expr.Children[0], env, noSkip, pos)
	case ExprSeq:
		// Evaluate every element into local (often stack-allocated) storage
		// first, then bump-copy the finished pointers into childSlab in one
		// shot. We can't reserve our range in childSlab up front and fill it
		// element-by-element: each recursive evalExpr call below may itself
		// bump-append its own children into the same shared slab, which
		// would land in the middle of our reserved range and corrupt it.
		var buf [8]*Node
		local := buf[:0]
		cur := pos
		for _, c := range expr.Children {
			node, next, ok := e.evalExpr(c, env, noSkip, cur)
			if !ok {
				return nil, 0, false
			}
			local = append(local, node)
			cur = next
		}
		return e.newNode(Node{Kind: KindSeq, Start: pos, End: cur, Children: e.newChildren(local)}), cur, true
	case ExprChoice:
		for i, alt := range expr.Children {
			node, next, ok := e.evalExpr(alt, env, noSkip, pos)
			if ok {
				return e.newNode(Node{Kind: KindChoice, Alt: i, Start: pos, End: next, Children: e.oneChild(node)}), next, true
			}
		}
		return nil, 0, false
	case ExprOptional:
		node, next, ok := e.evalExpr(expr.Children[0], env, noSkip, pos)
		if !ok {
			return e.newNode(Node{Kind: KindOptional, Start: pos, End: pos}), pos, true
		}
		return e.newNode(Node{Kind: KindOptional, Start: pos, End: next, Children: e.oneChild(node)}), next, true
	case ExprStar, ExprPlus:
		// Final count isn't known up front (it depends on how many times
		// the body matches), so this builds its own slice by ordinary
		// append growth rather than bump-copying into childSlab — sharing
		// the slab would just add a redundant copy for no benefit here.
		var children []*Node
		cur := pos
		for {
			node, next, ok := e.evalExpr(expr.Children[0], env, noSkip, cur)
			if !ok {
				break
			}
			children = append(children, node)
			zeroWidth := next == cur
			cur = next
			if zeroWidth {
				break
			}
		}
		if expr.Kind == ExprPlus && len(children) == 0 {
			return nil, 0, false
		}
		return e.newNode(Node{Kind: KindRepeat, Start: pos, End: cur, Children: children}), cur, true
	case ExprNot:
		_, _, ok := e.evalExpr(expr.Children[0], env, noSkip, pos)
		if ok {
			e.fail(pos)
			return nil, 0, false
		}
		return e.newNode(Node{Kind: KindNot, Start: pos, End: pos}), pos, true
	case ExprToken:
		return e.evalExpr(expr.Children[0], env, true, pos)
	default:
		panic(fmt.Sprintf("peg: unhandled expr kind %v", expr.Kind))
	}
}

func (e *evaluator) matchLiteral(lit string, noSkip bool, pos int) (int, bool) {
	p := pos
	if !noSkip {
		p = e.skipPos(p)
	}
	if p+len(lit) > len(e.input) || !equalFoldASCII(e.input[p:p+len(lit)], lit) {
		e.fail(p)
		return 0, false
	}
	end := p + len(lit)
	if isWordLiteral(lit) && end < len(e.input) && identCont(e.input[end]) {
		e.fail(p)
		return 0, false
	}
	return end, true
}

func isWordLiteral(s string) bool {
	return len(s) > 0 && identStart(s[0])
}

// identStart/identCont/equalFoldASCII back the case-insensitive
// keyword-literal boundary check above (so the literal 'IN' doesn't match
// a prefix of the input "INDEX"). Deliberately generic — this package has
// no notion of "SQL identifier"; it just needs *a* reasonable word-boundary
// heuristic for literal matching, independent of Task 2's dslIsIdentStart/
// dslIsIdentCont (which tokenize the unrelated `.gram` DSL) and of Task 4's
// real identifier primitive (which is what actually parses identifiers).
func identStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func identCont(c byte) bool {
	return identStart(c) || (c >= '0' && c <= '9')
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
