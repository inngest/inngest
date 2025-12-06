import { createFileRoute } from "@tanstack/react-router";
import { auth } from "@clerk/tanstack-react-start/server";
import { z } from "zod";
import { inngest } from "@/lib/inngest/client";

//
// Zod schema for UserMessage
const userMessageSchema = z.object({
  id: z.string().uuid("Valid message ID is required"),
  content: z.string().min(1, "Message content is required"),
  role: z.literal("user"),
  state: z.record(z.unknown()).optional(),
  clientTimestamp: z.string().optional(),
  systemPrompt: z.string().optional(),
});

//
// Zod schema for request body validation
const chatRequestSchema = z.object({
  userMessage: userMessageSchema,
  threadId: z.string().uuid().optional(),
  userId: z.string(),
  channelKey: z.string().optional(),
  history: z.array(z.any()).optional(),
});

export const Route = createFileRoute("/api/chat")({
  server: {
    handlers: {
      POST: async ({ request }) => {
        try {
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

          const body = await request.json();

          //
          // Validate request body with Zod
          const validationResult = chatRequestSchema.safeParse(body);
          if (!validationResult.success) {
            return new Response(
              JSON.stringify({
                error:
                  validationResult.error.errors[0]?.message ??
                  "Invalid request",
              }),
              {
                status: 400,
                headers: { "Content-Type": "application/json" },
              },
            );
          }

          const {
            userMessage,
            threadId: providedThreadId,
            channelKey,
            history,
          } = validationResult.data;

          //
          // Channel-first validation: require either userId OR channelKey
          if (!userId && !channelKey) {
            return new Response(
              JSON.stringify({
                error: "Either userId or channelKey is required",
              }),
              {
                status: 400,
                headers: { "Content-Type": "application/json" },
              },
            );
          }

          //
          // If the client didn't provide a threadId, omit generation here.
          // AgentKit will create one during initializeThread; the canonical ID will
          // be returned in the response from this route.
          const threadId = providedThreadId || undefined;

          //
          // Send event to Inngest to trigger the agent chat
          await inngest.send({
            name: "insights-agent/chat.requested",
            data: {
              threadId: threadId ?? undefined,
              history,
              userMessage,
              userId,
              channelKey,
            },
          });

          return new Response(
            JSON.stringify({
              success: true,
              threadId: threadId,
            }),
            {
              status: 200,
              headers: { "Content-Type": "application/json" },
            },
          );
        } catch (error) {
          return new Response(
            JSON.stringify({
              error:
                error instanceof Error ? error.message : "Failed to start chat",
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
