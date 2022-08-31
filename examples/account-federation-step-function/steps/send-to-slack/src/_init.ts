import type { EventTriggers, Args } from "./types";

/**
 * Init initializes the context for running the function.  This calls
 * start() when
 */
async function init() {
  let context: Args | undefined;

  // Import this asynchronously, such that any top-level 
  // errors in user code are caught.
  const { run } = await import("./index");

  // We pass the event in as an argument to the node function.  Running
  // npx ts-node "./foo.bar" means we have 2 arguments prior to the event.
  // We'll also be adding stdin and lambda compatibility soon.
  context = JSON.parse(process.argv[2]);

  if (!context) {
    throw new Error("unable to parse context");
  }

  const result = await run(context);
  return result;
}

init()
  .then((body) => {
    if (typeof body === "string") {
      console.log(JSON.stringify({ body }));
      return;
    }
    console.log(JSON.stringify(body));
  })
  .catch((e: Error) => {
    // TODO: Log error and stack trace.
    console.log(
      JSON.stringify({
        error: e.message || e.stack,
        stack: e.stack,
        status: 500,
      })
    );
    process.exit(1);
  });
