/**
 * A `step.invoke` shows the run it started, inside itself.
 *
 * The thing worth guarding is that the layout and the node agree about how much
 * room the child needs. They are computed in two different files at two
 * different times — the graph reserves the space before React ever renders the
 * node — so if they ever disagree the child is either clipped or floats in an
 * empty box, and neither shows up in a unit test unless it is asked for here.
 */
import { describe, expect, it } from 'vitest';

import type { Trace } from '../types';
import { traceRollup } from '../utils/traceConversion';
import child from './__fixtures__/child.json';
import invoke from './__fixtures__/invoke.json';
import { toCanvasGraph } from './graph';
import { LAYOUT, invokeOpenHeight, toFlowElements } from './toFlowElements';

function graphOf(fixture: unknown) {
  return toCanvasGraph(traceRollup((fixture as { run: { trace: Trace } }).run.trace));
}

const parent = graphOf(invoke);
const childGraph = graphOf(child);

const invokeNode = parent.nodes.find((n) => n.kind === 'invoke');

describe('an invoke that can be opened', () => {
  it('carries the child run it started', () => {
    // Without this the disclosure has nothing to fetch, and every assertion
    // below would be vacuously true.
    expect(invokeNode).toBeDefined();
    expect(invokeNode?.childRunID).toBeTruthy();
  });

  it('is an ordinary node until someone opens it', () => {
    const { nodes } = toFlowElements(parent);
    const node = nodes.find((n) => n.id === invokeNode!.id);

    expect(node?.height).toBe(LAYOUT.nodeHeight);
    expect(node?.data.childOpen).toBeUndefined();
  });

  it('offers no disclosure at all without a loader', () => {
    // An affordance that does nothing is worse than none: the chevron only
    // appears when `onToggleChild` is present to answer it.
    const { nodes } = toFlowElements(parent);
    expect(nodes.find((n) => n.id === invokeNode!.id)?.data.onToggleChild).toBeUndefined();
  });

  it('grows to exactly the room the node will draw into', () => {
    const { nodes } = toFlowElements(parent, undefined, undefined, undefined, {
      open: new Set([invokeNode!.id]),
      graphs: new Map([[invokeNode!.childRunID!, childGraph]]),
      loading: new Set(),
      onToggle: () => {},
    });

    const node = nodes.find((n) => n.id === invokeNode!.id);
    expect(node?.data.childOpen).toBe(true);
    expect(node?.data.childGraph).toBe(childGraph);

    // The one invariant that matters: the layout reserved what the node draws.
    expect(node?.height).toBe(invokeOpenHeight(childGraph.levels.length));
    expect(node?.height).toBeGreaterThan(LAYOUT.nodeHeight);
  });

  it('reserves room while the child is still loading', () => {
    // Opening one must not make the graph jump twice — once on open and again
    // when the child lands. It takes a default height immediately.
    const { nodes } = toFlowElements(parent, undefined, undefined, undefined, {
      open: new Set([invokeNode!.id]),
      graphs: new Map(),
      loading: new Set([invokeNode!.childRunID!]),
      onToggle: () => {},
    });

    const node = nodes.find((n) => n.id === invokeNode!.id);
    expect(node?.data.childLoading).toBe(true);
    expect(node?.height).toBeGreaterThan(LAYOUT.nodeHeight);
  });

  it('stays within bounds however deep the child is', () => {
    // A deep child scrolls inside its own region rather than pushing the graph
    // around without limit.
    expect(invokeOpenHeight(500)).toBeLessThanOrEqual(LAYOUT.invokeOpenMaxHeight);
    expect(invokeOpenHeight(0)).toBeGreaterThanOrEqual(LAYOUT.invokeOpenMinHeight);
    expect(invokeOpenHeight(undefined)).toBeGreaterThanOrEqual(LAYOUT.invokeOpenMinHeight);
  });

  it('does not move anything else on the level when it opens', () => {
    // The invoke sits alone on its level in this shape, so opening it must not
    // shift its neighbours — a disclosure that reflows the whole graph costs
    // the reader the place they were looking at.
    const closed = toFlowElements(parent);
    const open = toFlowElements(parent, undefined, undefined, undefined, {
      open: new Set([invokeNode!.id]),
      graphs: new Map([[invokeNode!.childRunID!, childGraph]]),
      loading: new Set(),
      onToggle: () => {},
    });

    for (const before of closed.nodes) {
      if (before.id === invokeNode!.id) continue;
      const after = open.nodes.find((n) => n.id === before.id);
      expect(after?.position, `${before.id} moved`).toEqual(before.position);
    }
  });
});
