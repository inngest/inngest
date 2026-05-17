import { inngestV4 } from "@/inngest/client_v4";

export const POST = inngestV4.endpoint(async () => {
  return new Response(JSON.stringify({ hello: "world" }), {
    headers: { "content-type": "application/json" },
  });
});
