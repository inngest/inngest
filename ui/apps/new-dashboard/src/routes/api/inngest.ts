import { serve } from "inngest/edge";
import { inngest } from "@/data/inngest/client";
import { runAgentNetwork } from "@/data/inngest/functions/run-network";
import { createFileRoute } from "@tanstack/react-router";

const handler = serve({ client: inngest, functions: [runAgentNetwork] });

export const Route = createFileRoute("/api/inngest")({
  server: {
    handlers: {
      GET: async ({ request }) => handler(request),
      POST: async ({ request }) => handler(request),
      PUT: async ({ request }) => handler(request),
    },
  },
});
