import { run } from "./index";

/**
 * init is a basic wrapper function that parses the command line arguments and
 * calls the main function, handling and logging the results or errors.
 */
async function init() {
  const context = JSON.parse(process.argv[2]);
  if (!context) {
    throw new Error("unable to parse context");
  }
  return await run(context);
}

init()
  .then((body) => {
    if (typeof body === "string") {
      console.log(JSON.stringify({ body }));
      return;
    }
    console.log(JSON.stringify(body));
  })
  .catch((e) => {
    console.log(
      JSON.stringify({
        error: e.stack || e.message,
        status: 500,
      })
    );
    process.exit(1);
  });
