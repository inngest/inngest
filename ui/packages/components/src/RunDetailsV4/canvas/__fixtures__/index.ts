/**
 * Every committed fixture, in one list, so something other than a test can walk
 * them — specifically the dev-only fixture gallery in dev-server-ui, which
 * renders each one through the real Canvas and Timeline.
 *
 * Static imports rather than `import.meta.glob`: this package is also consumed
 * by the Cloud dashboard, and a glob is a bundler feature not every consumer
 * has. The cost is that a new fixture has to be added here by hand, so
 * `index.test.ts` fails if one is left out.
 *
 * `hasLineage` counts spans carrying `parentStepIDs` — the SDK reporting which
 * steps each new step waited for. It is the difference between a fixture that
 * exercises `resolveBranches` and one that only exercises the fallback, and it
 * is measured from the payload rather than asserted, so it cannot drift.
 */
import blocked from './blocked.json';
import cancelled from './cancelled.json';
import chains from './chains.json';
import child from './child.json';
import emit from './emit.json';
import failcluster from './failcluster.json';
import failure from './failure.json';
import gnarlyDupeNames from './gnarly-dupe-names.json';
import gnarlyDynamic from './gnarly-dynamic.json';
import gnarlyForeignAsync from './gnarly-foreign-async.json';
import gnarlyMixed from './gnarly-mixed.json';
import gnarlyNestedInBranch from './gnarly-nested-in-branch.json';
import gnarlyNested from './gnarly-nested.json';
import gnarlyRace from './gnarly-race.json';
import gnarlySleepBranch from './gnarly-sleep-branch.json';
import gnarlyUnbalanced from './gnarly-unbalanced.json';
import inflight from './inflight.json';
import invoke from './invoke.json';
import longgap from './longgap.json';
import loop40 from './loop40.json';
import loop from './loop.json';
import noretry from './noretry.json';
import parallelInferred from './parallel-inferred.json';
import parallel from './parallel.json';
import retry from './retry.json';
import simple from './simple.json';
import step from './step.json';
import t19chains from './t19-chains.json';
import t19deadend from './t19-deadend.json';
import t19nested from './t19-nested.json';
import t19parallel from './t19-parallel.json';
import t19race from './t19-race.json';
import tall500 from './tall500.json';
import v4branchesBalanced from './v4branches-balanced.json';
import v4branchesRagged from './v4branches-ragged.json';
import v4chains from './v4chains.json';
import v4deep3 from './v4deep3.json';
import v4parallel from './v4parallel.json';
import v4pathological from './v4pathological.json';
import v4sequential from './v4sequential.json';
import wait from './wait.json';
import waitmatched from './waitmatched.json';
import waittimeout from './waittimeout.json';
import wide from './wide.json';

/** The raw GraphQL `data` for the `GetRun` query the UI issues. */
export type FixtureData = {
  run: { status: string; trace: unknown };
};

/**
 * Which client produced the run.
 *
 * This used to be the load-bearing distinction: a v3 client runs
 * ExecutionVersion.V1 and reports one opcode per response, so `plannedSteps` was
 * absent and grouping had to be inferred. That is no longer true. Once the
 * loader began lifting `plannedSteps` off discovery spans, v3 runs started
 * reporting exact groups too, and recapturing the fixtures against a current Dev
 * Server made every one of them 'sdk'.
 *
 * So the two code paths in `graph.ts` are now separated by whether a capture
 * predates that loader change, not by SDK version. `parallel-inferred` is the
 * one preserved pre-loader capture and the only thing exercising the fallback;
 * a test asserts that it still does.
 */
export type FixtureSDK = 'v3' | 'v4';

/** Loose grouping, only so the gallery can put like next to like. */
export type FixtureGroup =
  | 'basics'
  | 'waiting'
  | 'failure'
  | 'parallel'
  | 'adversarial'
  | 'lineage'
  | 'scale';

export type CanvasFixture = {
  /** Filename without the extension. Stable; used in the gallery's URL. */
  id: string;
  title: string;
  /** What the run does, and why this fixture is worth keeping. */
  note: string;
  sdk: FixtureSDK;
  group: FixtureGroup;
  /**
   * Set when the payload was not captured from a real run. Says what was done
   * to it, because a synthetic fixture must never be mistaken for evidence.
   */
  synthetic?: string;
  data: FixtureData;
};

const asData = (raw: unknown) => raw as FixtureData;

export const FIXTURES: CanvasFixture[] = [
  // --- basics -------------------------------------------------------------
  {
    id: 'simple',
    title: 'No steps',
    note: 'A function with no steps at all. The degenerate case: trigger, then result.',
    sdk: 'v3',
    group: 'basics',
    data: asData(simple),
  },
  {
    id: 'step',
    title: 'Three steps with a sleep',
    note: 'step.run → step.sleep("2s") → step.run. The shape every layout rule is checked against first.',
    sdk: 'v3',
    group: 'basics',
    data: asData(step),
  },
  {
    id: 'v4sequential',
    title: 'Sequential, v4 SDK',
    note: 'Sequential with a sleep on a client that could report parallel batches. Guards against the grouping code inventing parallelism where there is none.',
    sdk: 'v4',
    group: 'basics',
    data: asData(v4sequential),
  },
  {
    id: 'emit',
    title: 'Emits events from a step',
    note: 'A step.sendEvent fanning out to two events, each of which started a run. The one piece of lineage the platform does not label: the step reports as an ordinary RUN and the ids of what it sent live inside its output payload.',
    sdk: 'v4',
    group: 'basics',
    data: asData(emit),
  },
  {
    id: 'invoke',
    title: 'step.invoke',
    note: 'A step.invoke of a child function between two steps. Carries the child run ID, so it is the fixture for expanding a child in place.',
    sdk: 'v3',
    group: 'basics',
    data: asData(invoke),
  },
  {
    id: 'child',
    title: 'The invoked child run',
    note: 'The run that `invoke` started, captured separately. Pair with `invoke` when working on inline child expansion.',
    sdk: 'v3',
    group: 'basics',
    data: asData(child),
  },

  // --- waiting ------------------------------------------------------------
  {
    id: 'wait',
    title: 'waitForEvent, timed out',
    note: 'A waitForEvent that was never satisfied. The run still completes, so waiting and failing must not look alike.',
    sdk: 'v3',
    group: 'waiting',
    data: asData(wait),
  },
  {
    id: 'waitmatched',
    title: 'waitForEvent, matched',
    note: 'A waitForEvent satisfied by a real event. The counterpart to `wait`: same node, opposite outcome.',
    sdk: 'v3',
    group: 'waiting',
    data: asData(waitmatched),
  },
  {
    id: 'waittimeout',
    title: 'waitForEvent, short timeout',
    note: 'A short timeout that expired. The wait dominates the run duration, which is what an elastic axis has to cope with.',
    sdk: 'v3',
    group: 'waiting',
    data: asData(waittimeout),
  },
  {
    id: 'blocked',
    title: 'Held by a concurrency limit',
    note: 'The losing half of two runs contending on `concurrency: { limit: 1 }`. It spent 6.3s queued before executing, so the queued phase is real rather than the usual ~0ms and the waiting-vs-working encoding has something to show.',
    sdk: 'v4',
    group: 'waiting',
    data: asData(blocked),
  },
  {
    id: 'longgap',
    title: 'Seven days asleep',
    note: 'Seven days elapsed, a fraction of a second executing. The case a linear axis cannot draw: at true scale every step is sub-pixel.',
    sdk: 'v3',
    group: 'waiting',
    synthetic:
      'Generated by `node make-longgap.mjs`, which takes the real `step.json` capture and pushes every timestamp at or after the sleep forward by seven days. Only the clock is changed; span shapes and work durations are untouched. A long sleep cannot be captured by waiting for it, and hand-writing the payload would invent span artefacts nobody would think to invent.',
    data: asData(longgap),
  },

  // --- failure ------------------------------------------------------------
  {
    id: 'retry',
    title: 'Failed once, then succeeded',
    note: 'A step that failed and was retried into success. The run reads as green; the recovery is the thing worth surfacing.',
    sdk: 'v3',
    group: 'failure',
    data: asData(retry),
  },
  {
    id: 'noretry',
    title: 'NonRetriableError, caught',
    note: 'A step fails permanently but userland catches it, so the run succeeds. A red step inside a green run.',
    sdk: 'v3',
    group: 'failure',
    data: asData(noretry),
  },
  {
    id: 'failure',
    title: 'Run failed outright',
    note: 'A successful step, then one that takes the run down with it. The only FAILED capture in the set.',
    sdk: 'v3',
    group: 'failure',
    data: asData(failure),
  },
  {
    id: 'gnarly-mixed',
    title: 'One branch failed, run survived',
    note: 'A failing branch alongside a succeeding one, caught. Checks that one red node does not colour its whole level.',
    sdk: 'v4',
    group: 'failure',
    data: asData(gnarlyMixed),
  },
  {
    id: 'cancelled',
    title: 'Cancelled mid-flight',
    note: 'A run cancelled while a waitForEvent was still open, so a step is left WAITING inside a CANCELLED run. Neither succeeded nor failed — colouring it as either would be a lie, which is why this shape needed capturing.',
    sdk: 'v4',
    group: 'failure',
    data: asData(cancelled),
  },

  // --- parallel -----------------------------------------------------------
  {
    id: 'parallel',
    title: 'Promise.all of three, then one',
    note: 'A three-wide fan-out with grouping inferred from execution overlap, because the v3 client cannot report it.',
    sdk: 'v3',
    group: 'parallel',
    data: asData(parallel),
  },
  {
    id: 'parallel-inferred',
    title: 'The same fan-out, with nothing reported',
    note: 'Kept from before the loader began lifting `plannedSteps` off discovery spans. Once it did, even v1-SDK runs started reporting exact groups — so this is now the only capture in the set that exercises the inference fallback at all, and the "grouping: inferred" badge with it.',
    sdk: 'v3',
    group: 'parallel',
    data: asData(parallelInferred),
  },
  {
    id: 'v4parallel',
    title: 'Promise.all of three, v4 SDK',
    note: 'The same shape with `plannedSteps` present, so the grouping is exact rather than inferred. The two together are the A/B for that code path.',
    sdk: 'v4',
    group: 'parallel',
    data: asData(v4parallel),
  },
  {
    id: 'chains',
    title: 'Parallel branches that are chains',
    note: 'Promise.all over two two-step branches, then a join. Branch membership matters here: the wrong pairing crosses the branches.',
    sdk: 'v3',
    group: 'parallel',
    data: asData(chains),
  },
  {
    id: 'v4chains',
    title: 'Parallel chains, v4 SDK',
    note: 'Two plans — [left-1, right-1] then [left-2, right-2] — so the level structure is stated by the SDK rather than guessed.',
    sdk: 'v4',
    group: 'parallel',
    data: asData(v4chains),
  },
  {
    id: 'inflight',
    title: 'Captured mid-run',
    note: 'The only RUNNING capture: one parallel branch has landed, the other has not. Everything that assumes an end time has to cope with this.',
    sdk: 'v3',
    group: 'parallel',
    data: asData(inflight),
  },

  // --- adversarial --------------------------------------------------------
  {
    id: 'gnarly-unbalanced',
    title: 'Branches of unequal depth',
    note: 'Levelling assumes each discovery plans one row; an unbalanced fan-out is the first thing that should break that assumption.',
    sdk: 'v4',
    group: 'adversarial',
    data: asData(gnarlyUnbalanced),
  },
  {
    id: 'gnarly-nested',
    title: 'Nested Promise.all',
    note: 'A fan-out whose branches are themselves fan-outs.',
    sdk: 'v4',
    group: 'adversarial',
    data: asData(gnarlyNested),
  },
  {
    id: 'gnarly-race',
    title: 'Promise.race',
    note: 'The race path skips discovery coalescing, so every branch schedules its own discovery. The losers are drawn as broken edges, not dependencies.',
    sdk: 'v4',
    group: 'adversarial',
    data: asData(gnarlyRace),
  },
  {
    id: 'gnarly-sleep-branch',
    title: 'A sleep inside one branch',
    note: 'One branch sleeps while the other keeps working, so a wait node has to sit inside a parallel level.',
    sdk: 'v4',
    group: 'adversarial',
    data: asData(gnarlySleepBranch),
  },
  {
    id: 'gnarly-dupe-names',
    title: 'The same step name twice',
    note: 'The same name in both branches. The SDK disambiguates with a :1/:2 suffix assigned in discovery order — which is not stable across runs, so nothing may depend on it.',
    sdk: 'v4',
    group: 'adversarial',
    data: asData(gnarlyDupeNames),
  },
  {
    id: 'gnarly-foreign-async',
    title: 'Steps behind non-Inngest async work',
    note: 'Steps discovered only after unrelated awaits, which is where response ordering stops reflecting branch structure.',
    sdk: 'v4',
    group: 'adversarial',
    data: asData(gnarlyForeignAsync),
  },
  {
    id: 'gnarly-dynamic',
    title: 'Fan-out width from a step output',
    note: 'The width is decided by the previous step, so nothing static could have predicted it.',
    sdk: 'v4',
    group: 'adversarial',
    data: asData(gnarlyDynamic),
  },
  {
    id: 'gnarly-nested-in-branch',
    title: 'A branch that fans out',
    note: 'One resumption discovers two steps, so server-side "k-th most recent completion" attribution stops lining up. The client is expected to notice and decline rather than guess.',
    sdk: 'v4',
    group: 'adversarial',
    data: asData(gnarlyNestedInBranch),
  },

  // --- lineage ------------------------------------------------------------
  // Captured after the SDK began reporting which steps each new step waited
  // for. These carry `parentStepIDs`; the `gnarly-*` set above predates it.
  {
    id: 't19-parallel',
    title: 'Fan-out, with reported lineage',
    note: 'The same fan-out as `v4parallel`, recaptured once the SDK reported parent sets.',
    sdk: 'v4',
    group: 'lineage',
    data: asData(t19parallel),
  },
  {
    id: 't19-chains',
    title: 'Chains, with reported lineage',
    note: 'Parallel chains where every hop is stated rather than inferred. The densest lineage capture at this size.',
    sdk: 'v4',
    group: 'lineage',
    data: asData(t19chains),
  },
  {
    id: 't19-nested',
    title: 'Nested combinators, with lineage',
    note: 'Nested Promise.all with parent sets reported.',
    sdk: 'v4',
    group: 'lineage',
    data: asData(t19nested),
  },
  {
    id: 't19-race',
    title: 'Race, with lineage',
    note: 'A race where the winner is a real dependency and the losers are known alternates. The fixture behind the dashed-edge rendering.',
    sdk: 'v4',
    group: 'lineage',
    data: asData(t19race),
  },
  {
    id: 't19-deadend',
    title: 'A dead-end branch',
    note: 'Two steps started, only one ever awaited. The other is a genuine dead end, and must be left dangling rather than joined to something.',
    sdk: 'v4',
    group: 'lineage',
    data: asData(t19deadend),
  },
  {
    id: 'v4pathological',
    title: 'Pathological chains, graded',
    note: 'Ground truth is encoded in the step names — a step called `c<=a,b` declares the parents it should be given — so attribution can be graded automatically.',
    sdk: 'v4',
    group: 'lineage',
    data: asData(v4pathological),
  },
  {
    id: 'v4deep3',
    title: 'Three-deep symmetric chains',
    note: 'Jittered work so completion order varies between runs. Currently imported by no test — it is here to be looked at.',
    sdk: 'v4',
    group: 'lineage',
    data: asData(v4deep3),
  },

  // --- scale --------------------------------------------------------------
  {
    id: 'loop',
    title: 'Agent loop, 8 iterations',
    note: 'Eight sequential iterations of the same two-step shape.',
    sdk: 'v3',
    group: 'scale',
    data: asData(loop),
  },
  {
    id: 'loop40',
    title: 'Agent loop, 40 iterations',
    note: '80 steps in 80 levels. Committed precisely because it breaks the view: fitted, it is a grey hairline. The reason collapsing exists.',
    sdk: 'v3',
    group: 'scale',
    data: asData(loop40),
  },
  {
    id: 'tall500',
    title: '500 sequential steps',
    note: '500 steps, one per level, each a few milliseconds. The tall case: nothing here is individually interesting, and at 28px a row it is 14,000px of scrolling to find out.',
    sdk: 'v4',
    group: 'scale',
    data: asData(tall500),
  },
  {
    id: 'failcluster',
    title: 'A failure cluster in a long run',
    note: '500 steps with 21 consecutive failures two thirds of the way through. The density strip exists to make exactly this visible in the first screenful with no interaction, and this is the fixture that test is run against.',
    sdk: 'v4',
    group: 'scale',
    synthetic:
      "Generated by `node make-failcluster.mjs`, which rewrites the status of a contiguous run of steps in the real `tall500.json` capture. Only status and attempts change; timestamps, names and span structure are the capture's. Producing this for real needs a fixture app that can fail on demand at a chosen iteration.",
    data: asData(failcluster),
  },
  {
    id: 'wide',
    title: '12-wide fan-out',
    note: 'Twelve parallel steps then a collect. Wide rather than tall — the other way a run stops being readable.',
    sdk: 'v3',
    group: 'scale',
    data: asData(wide),
  },
  {
    id: 'v4branches-ragged',
    title: 'Stress: ragged branches',
    note: 'The parameterised stress shape with uneven branch depths, so completion order and plan order disagree.',
    sdk: 'v4',
    group: 'scale',
    data: asData(v4branchesRagged),
  },
  {
    id: 'v4branches-balanced',
    title: 'Stress: balanced branches',
    note: 'The widest lineage capture in the set: 48 steps, all with reported parents.',
    sdk: 'v4',
    group: 'scale',
    data: asData(v4branchesBalanced),
  },
];

export const FIXTURES_BY_ID: ReadonlyMap<string, CanvasFixture> = new Map(
  FIXTURES.map((f) => [f.id, f])
);
