import { createFileRoute } from "@tanstack/react-router";
import { auth } from "@clerk/tanstack-react-start/server";
import { getSubscriptionToken } from "@inngest/realtime";
import { inngest } from "@/data/inngest/client";
import { createChannel } from "@/data/inngest/realtime";

export type RequestBody = {
  userId?: string;
  channelKey?: string;
};

export const Route = createFileRoute("/api/realtime/token")({
  server: {
    handlers: {
      POST: async ({ request }) => {
        //
        // Authenticate the user using Clerk
        const { userId } = await auth();
        if (!userId) {
          return new Response(
            JSON.stringify({ error: "Please sign in to create a token" }),
            {
              status: 401,
              headers: { "Content-Type": "application/json" },
            },
          );
        }

        try {
          //
          // Get the channel key from the request body and validate it
          const { channelKey } = (await request.json()) as RequestBody;
          if (!channelKey) {
            return new Response(
              JSON.stringify({ error: "channelKey is required" }),
              {
                status: 400,
                headers: { "Content-Type": "application/json" },
              },
            );
          }

          //
          // Create a subscription token for the resolved channel
          // Match publisher semantics: when channelKey is provided, we publish to that key directly.
          const token = await getSubscriptionToken(inngest, {
            channel: createChannel(channelKey),
            topics: ["agent_stream"],
          });

          return new Response(JSON.stringify(token), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        } catch (error) {
          return new Response(
            JSON.stringify({
              error:
                error instanceof Error
                  ? error.message
                  : "Failed to create subscription token",
            }),
            {
              status: 500,
              headers: { "Content-Type": "application/json" },
            },
          );
        }
      },
    },
  },
});
