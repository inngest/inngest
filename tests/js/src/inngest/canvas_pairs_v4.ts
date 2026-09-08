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
