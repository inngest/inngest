// The fixture shapes, each defined once and run BOTH ways.
//
// v4 checkpoints a `step.run` it can execute inline: the step is reported out
// of band while the request is still open, so the SDK carries on into the next
// step and one request covers work that would otherwise take several. That
// changes the trace enough that the two modes are different pictures of the
// same function, which is the comparison the fixtures page exists to make.
//
// So each shape is written once, here, and instantiated on two clients that
// differ in exactly one setting. Anything that diverges between the two
// captures is the checkpointing and nothing else.
import { Inngest, step, type InngestFunction } from "inngest-v4";

const base = {
  isDev: true,
  baseUrl: process.env.INNGEST_BASE_URL ?? "http://127.0.0.1:8288",
} as const;

export const cpClient = new Inngest({ id: "canvas-cp", ...base, checkpointing: true });
export const noCpClient = new Inngest({ id: "canvas-nocp", ...base, checkpointing: false });

/** A shape: one handler, one id, one event, run on both clients. */
type Shape = {
  id: string;
  handler: () => Promise<unknown>;
  /** Extra function config, for the shapes that need it. */
  opts?: Record<string, unknown>;
};

const SHAPES: Shape[] = [
  // No steps at all. The whole run is the request that asks what to do and is
  // told nothing, so the trace has to be quiet here or it is noise everywhere.
  { id: "simple", handler: async () => "done" },

  // A step, a sleep, a step. The sleep cannot be run inline whichever mode we
  // are in, so this is the shape where checkpointing shows most plainly: with
  // it the first step and the sleep share a request, without it they do not.
  {
    id: "sequential",
    handler: async () => {
      await step.run("first step", async () => "first");
      await step.sleep("for 2s", "2s");
      await step.run("second step", async () => "second");
      return "done";
    },
  },

  // Three steps in a row with nothing between them: every one of them is
  // inline-able, so checkpointing collapses the whole run into one request.
  {
    id: "emit",
    handler: async () => {
      await step.run("prepare", async () => "ready");
      await step.sendEvent("fan-out", [
        { name: "tests/pair.emitted", data: { n: 1 } },
        { name: "tests/pair.emitted", data: { n: 2 } },
      ]);
      await step.run("after", async () => "done");
      return "done";
    },
  },

  // A parallel batch has to be PLANNED -- the SDK cannot run three steps
  // inline and still report them as parallel -- so the fan-out looks the same
  // either way and only the sequential step after it moves.
  {
    id: "parallel",
    handler: async () => {
      await Promise.all([
        step.run("a", async () => "a"),
        step.run("b", async () => "b"),
        step.run("c", async () => "c"),
      ]);
      await step.run("d", async () => "d");
      return "done";
    },
  },

  // Parallel branches that are themselves chains: the branch membership case,
  // and the one where planning and checkpointing appear in the same run.
  {
    id: "chains",
    handler: async () => {
      const chain = async (name: string) => {
        const one = await step.run(`${name}-1`, async () => 1);
        return step.run(`${name}-2`, async () => (one as number) + 1);
      };
      const [x, y] = await Promise.all([chain("left"), chain("right")]);
      await step.run("join", async () => (x as number) + (y as number));
      return { x, y };
    },
  },

  // Branches of different depths: levelling assumes each discovery plans one
  // row, and an unbalanced fan-out is the first thing that breaks that.
  {
    id: "unbalanced",
    handler: async () => {
      const deep = async () => {
        const one = await step.run("deep-1", async () => 1);
        const two = await step.run("deep-2", async () => (one as number) + 1);
        return step.run("deep-3", async () => (two as number) + 1);
      };
      const shallow = async () => step.run("shallow-1", async () => 10);
      const [a, b] = await Promise.all([deep(), shallow()]);
      return { a, b };
    },
  },

  // A fan-out whose branches are themselves fan-outs.
  {
    id: "nested",
    handler: async () => {
      const [[a1, a2], b] = await Promise.all([
        Promise.all([step.run("a1", async () => 1), step.run("a2", async () => 2)]),
        step.run("b", async () => 3),
      ]);
      await step.run("collect", async () => (a1 as number) + (a2 as number) + (b as number));
      return "done";
    },
  },

  // A branch that itself fans out: one resumption discovers two steps.
  {
    id: "nested-in-branch",
    handler: async () => {
      const fanning = async () => {
        await step.run("f-1", async () => 1);
        await Promise.all([step.run("f-2a", async () => 2), step.run("f-2b", async () => 3)]);
      };
      const plain = async () => {
        await step.run("p-1", async () => 1);
        await step.run("p-2", async () => 2);
      };
      await Promise.all([fanning(), plain()]);
      return "done";
    },
  },

  // Promise.race: every branch schedules its own discovery, because racing
  // deliberately skips coalescing.
  {
    id: "race",
    handler: async () => {
      const nap = (n: number) => new Promise((r) => setTimeout(r, n));
      const winner = await Promise.race([
        step.run("fast", async () => { await nap(100); return "fast"; }),
        step.run("slow", async () => { await nap(4000); return "slow"; }),
      ]);
      await step.run("after-race", async () => winner);
      return { winner };
    },
  },

  // A failing branch beside a succeeding one, caught so the run completes. One
  // red row in a level must not colour the level.
  {
    id: "mixed",
    opts: { retries: 0 },
    handler: async () => {
      const results = await Promise.allSettled([
        step.run("will-succeed", async () => "ok"),
        step.run("will-fail", async () => { throw new Error("intentional"); }),
      ]);
      await step.run("after-mixed", async () => results.map((r) => r.status).join(","));
      return "done";
    },
  },

  // A sleep inside one branch while the other keeps working: a wait sitting
  // inside a parallel level.
  {
    id: "sleep-in-branch",
    handler: async () => {
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
    },
  },

  // The same step name in both branches, which the SDK resolves with a :1/:2
  // suffix. Nothing in the drawing may depend on that suffix.
  {
    id: "dupe-names",
    handler: async () => {
      const branch = async (n: number) => {
        await step.run("work", async () => n);
        return step.run("finish", async () => n * 2);
      };
      await Promise.all([branch(1), branch(2)]);
      return "done";
    },
  },

  // A step discovered only after non-Inngest async work, where response order
  // stops reflecting branch structure.
  {
    id: "foreign-async",
    handler: async () => {
      const nap = (n: number) => new Promise((r) => setTimeout(r, n));
      const delayed = async () => { await nap(250); return step.run("late", async () => "late"); };
      const prompt = async () => step.run("early", async () => "early");
      await Promise.all([delayed(), prompt()]);
      return "done";
    },
  },

  // Fan-out width decided by a previous step's output: nothing static predicted
  // the shape.
  {
    id: "dynamic",
    handler: async () => {
      const n = (await step.run("decide", async () => 4)) as number;
      await Promise.all(Array.from({ length: n }, (_, i) => step.run(`dyn-${i}`, async () => i)));
      await step.run("finish", async () => n);
      return { n };
    },
  },

  // Three symmetric chains, three deep, with jitter so completion order varies
  // between runs and branch membership cannot be read off timing.
  {
    id: "deep3",
    handler: async () => {
      const jitter = () => new Promise((r) => setTimeout(r, Math.random() * 60));
      const chain = async (name: string) => {
        for (const i of [1, 2, 3]) {
          await step.run(`${name}-${i}`, async () => { await jitter(); return i; });
        }
        return name;
      };
      await Promise.all([chain("alpha"), chain("beta"), chain("gamma")]);
      return "done";
    },
  },

  // Two steps started, one awaited: the other is a genuine dead end, with
  // nothing in the function following it.
  {
    id: "dead-end",
    handler: async () => {
      const pa = step.run("a", async () => 1);
      const pb = step.run("b-deadend", async () => 2);
      const av = (await pa) as number;
      await step.run("c-after-a", async () => av + 1);
      await pb;
      return "done";
    },
  },

  // A sequential loop: twelve steps of one shape, which is what a collapsed
  // group is for.
  {
    id: "loop",
    handler: async () => {
      for (let i = 0; i < 12; i++) await step.run(`poll-${i}`, async () => i);
      return "done";
    },
  },

  // A wide fan-out: twelve steps planned by one request.
  {
    id: "wide",
    handler: async () => {
      await Promise.all(Array.from({ length: 12 }, (_, i) => step.run(`worker-${i}`, async () => i)));
      await step.run("collect", async () => "collected");
      return "done";
    },
  },

  // A wait nothing satisfies, so it expires. The run carries on: a timeout is a
  // result the function can act on.
  {
    id: "wait-timeout",
    handler: async () => {
      await step.run("before", async () => "before");
      const got = await step.waitForEvent("for a signal", {
        event: "tests/pair.never",
        timeout: "3s",
      });
      await step.run("after", async () => (got ? "matched" : "timed out"));
      return "done";
    },
  },
];

/** The child `invoke` calls, one per client so each stays inside its own app. */
const child = (c: Inngest.Any) =>
  c.createFunction(
    { id: "pair-child", triggers: [{ event: "tests/pair.child" }] },
    async () => {
      await step.run("work", async () => "worked");
      await step.run("finish", async () => "finished");
      return "done";
    }
  );

/** The handler that receives what `emit` sends, so the events lead somewhere. */
const emitted = (c: Inngest.Any) =>
  c.createFunction(
    { id: "pair-emitted", triggers: [{ event: "tests/pair.emitted" }] },
    async () => {
      await step.run("handle", async () => "handled");
      return "done";
    }
  );

function build(c: Inngest.Any) {
  const kid = child(c);
  const fns: InngestFunction.Any[] = [kid, emitted(c)];
  for (const s of SHAPES) {
    fns.push(
      c.createFunction(
        { id: `pair-${s.id}`, triggers: [{ event: `tests/pair.${s.id}` }], ...(s.opts ?? {}) },
        s.handler
      )
    );
  }
  // `invoke` needs the child in scope, so it is built here rather than listed
  // above with the shapes that do not.
  fns.push(
    c.createFunction(
      { id: "pair-invoke", triggers: [{ event: "tests/pair.invoke" }] },
      async () => {
        await step.run("before", async () => "before");
        const res = await step.invoke("call child", { function: kid, data: {} });
        await step.run("after", async () => "after");
        return { res };
      }
    )
  );
  return fns;
}

export const cpFns = build(cpClient);
export const noCpFns = build(noCpClient);
