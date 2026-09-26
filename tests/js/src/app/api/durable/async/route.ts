import { step } from "inngest-v4";
import { inngestV4 } from "@/v4/client";

export const POST = inngestV4.endpoint(async () => {
  await step.sleep("brief-pause", "1s");
  return new Response(JSON.stringify({ hello: "async" }), {
    headers: { "content-type": "application/json" },
  });
});
