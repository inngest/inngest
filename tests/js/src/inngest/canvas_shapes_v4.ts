// Canvas fixture functions on the v4 SDK.
//
// Why a second copy of these shapes: the v3 client in `canvas_shapes.ts` runs
// ExecutionVersion.V1, which reports one opcode per response. That means
// `response.step.ops` never carries a parallel batch and the executor never
// coalesces fan-in. inngest-v4 prefers ExecutionVersion.V2, so these functions
// are the only way to find out what the trace looks like when the SDK actually
// plans steps together.
import { Inngest, step } from "inngest-v4";

export const inngestV4Fns = new Inngest({
  id: "canvas-v4",
  isDev: true,
  baseUrl: process.env.INNGEST_BASE_URL ?? "http://127.0.0.1:8288",
});

// Straight fan-out: three steps planned in one response, then a sequential step.
export const v4Parallel = inngestV4Fns.createFunction(
  { id: "v4-parallel", triggers: [{ event: "tests/v4.parallel" }] },
  async () => {
    await Promise.all([
      step.run("a", async () => "a"),
      step.run("b", async () => "b"),
      step.run("c", async () => "c"),
    ]);
    await step.run("d", async () => "d");
    return "done";
  }
);

// Parallel branches that are chains — the branch-membership case.
export const v4Chains = inngestV4Fns.createFunction(
  { id: "v4-chains", triggers: [{ event: "tests/v4.chains" }] },
  async () => {
    const chain = async (name: string) => {
      const one = await step.run(`${name}-1`, async () => 1);
      return step.run(`${name}-2`, async () => one + 1);
    };
    const [x, y] = await Promise.all([chain("left"), chain("right")]);
    await step.run("join", async () => x + y);
    return { x, y };
  }
);

// Sequential with a sleep, to check the V1 false-positive does not recur.
export const v4Sequential = inngestV4Fns.createFunction(
  { id: "v4-sequential", triggers: [{ event: "tests/v4.sequential" }] },
  async () => {
    await step.run("first step", async () => "first");
    await step.sleep("for 2s", "2s");
    await step.run("second step", async () => "second");
    return "done";
  }
);

// Slow parallel branches, so an in-flight snapshot catches one landing while
// the other is still running.
export const v4Slow = inngestV4Fns.createFunction(
  { id: "v4-slow", triggers: [{ event: "tests/v4.slow" }] },
  async () => {
    const sleep = (n: number) => new Promise((r) => setTimeout(r, n));
    await Promise.all([
      step.run("slow-a", async () => {
        await sleep(9000);
        return "a";
      }),
      step.run("slow-b", async () => {
        await sleep(3000);
        return "b";
      }),
    ]);
    await step.run("after", async () => "after");
    return "done";
  }
);

// ---------------------------------------------------------------------------
// Deliberately horrible shapes. These exist to find out where the canvas lies.
// ---------------------------------------------------------------------------

// Branches of different depths. Levelling assumes each discovery plans one
// "row"; an unbalanced fan-out is the first thing that should break that.
export const v4Unbalanced = inngestV4Fns.createFunction(
  { id: "v4-unbalanced", triggers: [{ event: "tests/v4.unbalanced" }] },
  async () => {
    const deep = async () => {
      const one = await step.run("deep-1", async () => 1);
      const two = await step.run("deep-2", async () => one + 1);
      return step.run("deep-3", async () => two + 1);
    };
    const shallow = async () => step.run("shallow-1", async () => 10);
    const [a, b] = await Promise.all([deep(), shallow()]);
    return { a, b };
  }
);

// Nested Promise.all: a fan-out whose branches are themselves fan-outs.
export const v4Nested = inngestV4Fns.createFunction(
  { id: "v4-nested", triggers: [{ event: "tests/v4.nested" }] },
  async () => {
    const [[a1, a2], b] = await Promise.all([
      Promise.all([
        step.run("a1", async () => 1),
        step.run("a2", async () => 2),
      ]),
      step.run("b", async () => 3),
    ]);
    await step.run("collect", async () => a1 + a2 + b);
    return "done";
  }
);

// Promise.race — the ParallelMode.Race path, which deliberately skips discovery
// coalescing, so every branch schedules its own discovery.
export const v4Race = inngestV4Fns.createFunction(
  { id: "v4-race", triggers: [{ event: "tests/v4.race" }] },
  async () => {
    const sleep = (n: number) => new Promise((r) => setTimeout(r, n));
    const winner = await Promise.race([
      step.run("fast", async () => {
        await sleep(100);
        return "fast";
      }),
      step.run("slow", async () => {
        await sleep(4000);
        return "slow";
      }),
    ]);
    await step.run("after-race", async () => winner);
    return { winner };
  }
);

// A failing branch alongside a succeeding one, caught so the run completes.
// Tests that one red node in a level does not colour the whole level.
export const v4MixedOutcomes = inngestV4Fns.createFunction(
  { id: "v4-mixed", retries: 0, triggers: [{ event: "tests/v4.mixed" }] },
  async () => {
    const results = await Promise.allSettled([
      step.run("will-succeed", async () => "ok"),
      step.run("will-fail", async () => {
        throw new Error("intentional");
      }),
    ]);
    await step.run("after-mixed", async () => results.map((r) => r.status).join(","));
    return "done";
  }
);

// A sleep inside one branch while the other branch keeps working. The sleep is
// a node, so this checks a wait can sit *inside* a parallel level.
export const v4SleepInBranch = inngestV4Fns.createFunction(
  { id: "v4-sleep-branch", triggers: [{ event: "tests/v4.sleepbranch" }] },
  async () => {
    const napping = async () => {
      await step.run("before-nap", async () => 1);
      await step.sleep("nap", "2s");
      return step.run("after-nap", async () => 2);
    };
    const busy = async () => {
      await step.run("busy-1", async () => 1);
      return step.run("busy-2", async () => 2);
    };
    await Promise.all([napping(), busy()]);
    return "done";
  }
);

// The same step name in both branches. The SDK resolves this with a `:1`/`:2`
// suffix in discovery order and warns about it — worth seeing what we render.
export const v4DupeNames = inngestV4Fns.createFunction(
  { id: "v4-dupe-names", triggers: [{ event: "tests/v4.dupes" }] },
  async () => {
    const branch = async (n: number) => {
      await step.run("work", async () => n);
      return step.run("finish", async () => n * 2);
    };
    await Promise.all([branch(1), branch(2)]);
    return "done";
  }
);

// Steps discovered only after non-Inngest async work. The investigation flagged
// this as the case where response ordering stops reflecting branch structure.
export const v4ForeignAsync = inngestV4Fns.createFunction(
  { id: "v4-foreign-async", triggers: [{ event: "tests/v4.foreign" }] },
  async () => {
    const sleep = (n: number) => new Promise((r) => setTimeout(r, n));
    const delayed = async () => {
      await sleep(250); // not an Inngest step — a plain timer
      return step.run("late", async () => "late");
    };
    const prompt = async () => step.run("early", async () => "early");
    await Promise.all([delayed(), prompt()]);
    return "done";
  }
);

// Fan-out width decided by a previous step's output.
export const v4Dynamic = inngestV4Fns.createFunction(
  { id: "v4-dynamic", triggers: [{ event: "tests/v4.dynamic" }] },
  async () => {
    const n = await step.run("decide", async () => 4);
    await Promise.all(
      Array.from({ length: n }, (_, i) => step.run(`dyn-${i}`, async () => i))
    );
    await step.run("finish", async () => n);
    return { n };
  }
);

// Three-deep symmetric chains, with jittered work so completion order varies
// between runs. Used to test whether branch membership can be recovered from
// (completion order) x (plannedSteps array order).
export const v4Deep3 = inngestV4Fns.createFunction(
  { id: "v4-deep3", triggers: [{ event: "tests/v4.deep3" }] },
  async () => {
    const jitter = () => new Promise((r) => setTimeout(r, Math.random() * 60));
    const chain = async (name: string) => {
      for (const i of [1, 2, 3]) {
        await step.run(`${name}-${i}`, async () => {
          await jitter();
          return i;
        });
      }
      return name;
    };
    await Promise.all([chain("alpha"), chain("beta"), chain("gamma")]);
    return "done";
  }
);

// Parameterised stress shape for branch-membership testing. Event data controls
// width, depth and jitter, so a harness can vary completion order deliberately
// rather than hoping timing wobbles on its own.
export const v4Stress = inngestV4Fns.createFunction(
  { id: "v4-stress", triggers: [{ event: "tests/v4.stress" }] },
  async ({ event }) => {
    const branches = Number(event.data?.branches ?? 3);
    const depth = Number(event.data?.depth ?? 3);
    const jitter = Number(event.data?.jitter ?? 80);
    // Ragged: give each branch a different depth, so some finish early and the
    // level widths stop matching. The pairing must decline, not guess.
    const ragged = Boolean(event.data?.ragged);

    const names = Array.from({ length: branches }, (_, i) => `br${i}`);
    const chain = async (name: string, len: number) => {
      for (let i = 1; i <= len; i++) {
        await step.run(`${name}-${i}`, async () => {
          await new Promise((r) => setTimeout(r, Math.random() * jitter));
          return i;
        });
      }
    };
    await Promise.all(
      names.map((n, i) => chain(n, ragged ? Math.max(1, depth - (i % depth)) : depth))
    );
    return { branches, depth, ragged };
  }
);

// A branch that itself fans out. This is the shape the server-side parent
// attribution cannot get right on its own: one resumption discovers two steps,
// so "the k-th most recent completion" stops lining up. The client is expected
// to notice and decline.
export const v4NestedInBranch = inngestV4Fns.createFunction(
  { id: "v4-nested-in-branch", triggers: [{ event: "tests/v4.nestedbranch" }] },
  async () => {
    const fanning = async () => {
      await step.run("f-1", async () => 1);
      // one resumption, two new steps
      await Promise.all([
        step.run("f-2a", async () => 2),
        step.run("f-2b", async () => 3),
      ]);
    };
    const plain = async () => {
      await step.run("p-1", async () => 1);
      await step.run("p-2", async () => 2);
    };
    await Promise.all([fanning(), plain()]);
    return "done";
  }
);

// ---------------------------------------------------------------------------
// Pathological promise chains, for testing branch attribution.
//
// Ground truth is encoded in the step names: a step called "c<=a,b" should be
// attributed to steps "a" and "b". That makes grading automatic even for shapes
// where the right answer is not obvious by eye.
// ---------------------------------------------------------------------------
export const v4Pathological = inngestV4Fns.createFunction(
  { id: "v4-pathological", triggers: [{ event: "tests/v4.pathological" }] },
  async () => {
    const sleep = (n: number) => new Promise((r) => setTimeout(r, n));

    // 1. A true join: `join` depends on BOTH a and b. Any single-parent answer
    //    is at best half right.
    const [a, b] = await Promise.all([
      step.run("a", async () => 1),
      step.run("b", async () => 2),
    ]);
    await step.run("join<=a,b", async () => a + b);

    // 2. A continuation that leaves the microtask window before reaching its
    //    next step: real IO between parent and child.
    await Promise.all([
      (async () => {
        await step.run("io-parent", async () => 1);
        await sleep(120); // macrotask — the window has long closed
        await step.run("io-child<=io-parent", async () => 2);
      })(),
      (async () => {
        await step.run("other-1", async () => 1);
        await step.run("other-2<=other-1", async () => 2);
      })(),
    ]);

    // 3. A promise chain deeper than the 100-microtask drain.
    await Promise.all([
      (async () => {
        await step.run("deep-parent", async () => 1);
        let p: Promise<void> = Promise.resolve();
        for (let i = 0; i < 400; i++) p = p.then(() => undefined);
        await p;
        await step.run("deep-child<=deep-parent", async () => 2);
      })(),
      (async () => {
        await step.run("shallow-1", async () => 1);
        await step.run("shallow-2<=shallow-1", async () => 2);
      })(),
    ]);

    // 4. Asymmetric join: one branch feeds two children.
    const [x] = await Promise.all([
      step.run("fork", async () => 1),
      step.run("bystander", async () => 2),
    ]);
    await Promise.all([
      step.run("fork-kid-1<=fork", async () => x + 1),
      step.run("fork-kid-2<=fork", async () => x + 2),
    ]);

    return "done";
  }
);

// Fan out to two steps but only ever await one of them, so the other is a
// genuine dead end: nothing in the function follows it.
export const v4DeadEnd = inngestV4Fns.createFunction(
  { id: "v4-dead-end", triggers: [{ event: "tests/v4.deadend" }] },
  async () => {
    const pa = step.run("a", async () => 1);
    const pb = step.run("b-deadend", async () => 2);

    const av = await pa;                       // only `a` is awaited here
    await step.run("c<=a", async () => av + 1); // so only `a` has a successor

    await pb;                                  // settle it, but nothing follows
    return "done";
  }
);

// ---------------------------------------------------------------------------
// Shapes the fixture set was missing. Each one exists because a view has to
// cope with it and nothing captured so far exercises it.
// ---------------------------------------------------------------------------

// A run that stays alive long enough to be cancelled. Earlier attempts to
// capture a cancelled run lost the race — the run finished before the cancel
// landed — so this one parks on a wait it will never satisfy, leaving a
// generous window. A step running when the run was cancelled is neither
// succeeded nor failed, and colouring it as either is a lie.
export const v4Cancel = inngestV4Fns.createFunction(
  { id: "v4-cancel", triggers: [{ event: "tests/v4.cancel" }] },
  async () => {
    await step.run("before", async () => "before");

    // Ten minutes, so cancelling by hand is unhurried.
    await step.waitForEvent("never arrives", {
      event: "tests/v4.cancel.never",
      timeout: "10m",
    });

    await step.run("after", async () => "after");
    return "done";
  }
);

// ~500 sequential fast steps. `wide.json` is 12 across; nothing in the set was
// tall, and tall is what makes the current view unreadable — one row each, all
// of them 28px, none of them interesting on their own.
export const v4Tall = inngestV4Fns.createFunction(
  { id: "v4-tall", triggers: [{ event: "tests/v4.tall" }] },
  async ({ event }) => {
    const count = (event.data as { count?: number })?.count ?? 500;

    // Deliberately sequential: each step is discovered only after the previous
    // one returns, which is what produces one level per step.
    for (let i = 0; i < count; i++) {
      await step.run(`s${i}`, async () => i);
    }

    return { count };
  }
);

// Two runs contending on a limit of one, so the second spends real time queued
// rather than executing. Without this the queued phase is always ~0ms and the
// bar states that distinguish waiting from working have no data behind them.
export const v4Contended = inngestV4Fns.createFunction(
  {
    id: "v4-contended",
    concurrency: { limit: 1 },
    triggers: [{ event: "tests/v4.contended" }],
  },
  async () => {
    await step.run("hold", async () => {
      await new Promise((r) => setTimeout(r, 6000));
      return "held";
    });
    await step.run("release", async () => "released");
    return "done";
  }
);
