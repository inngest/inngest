const defaultCommentBlock = `// Rename this webhook to give the events a unique name,
    // or use a field from the incoming event as the event name.`;

//
// XXX: our server-side JS AST parser does not like ES6 style functions.
export const createTransform = ({
  eventName = `"webhook/request.received"`,
  dataParam = "evt",
  commentBlock = defaultCommentBlock,
}): string => {
  return `// transform accepts the incoming JSON payload from your
// webhook and must return an object that is in the Inngest event format.
//
// The raw argument is the original stringified request body. This is useful
// when you want to perform HMAC validation within your Inngest functions.
function transform(evt, headers = {}, queryParams = {}, raw = "") {
  return {
    ${commentBlock}
    name: ${eventName},
    data: ${dataParam},
  };
};`;
};

export const defaultTransform = createTransform({});
