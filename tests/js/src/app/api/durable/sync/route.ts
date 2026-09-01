import { inngestV4 } from "@/v4/client";

export const POST = inngestV4.endpoint(async () => {
  return new Response(JSON.stringify({ hello: "world" }), {
    headers: { "content-type": "application/json" },
  });
});
