import { createFileRoute } from '@tanstack/react-router';
import { auth } from '@clerk/tanstack-react-start/server';
import { z } from 'zod/v3';
import { inngest } from '@/lib/inngest/client';

//
// The browser half of the agent's validate_query round trip: the chat UI runs
// the SQL with the user's own credentials and posts the outcome here, which
// forwards it as the event the agent loop is waiting on (step.waitForEvent).
const validationResultSchema = z.object({
  validationId: z.string().min(1).max(64),
  ok: z.boolean(),
  columns: z.array(z.string().max(256)).max(100).optional(),
  rowCount: z.number().int().nonnegative().optional(),
  diagnostics: z
    .array(
      z.object({
        code: z.string().max(128).optional(),
        message: z.string().max(2000),
      }),
    )
    .max(20)
    .optional(),
});

export const Route = createFileRoute('/api/chat-validate')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        try {
          const { userId } = await auth();
          if (!userId) {
            return new Response(JSON.stringify({ error: 'Please sign in' }), {
              status: 401,
              headers: { 'Content-Type': 'application/json' },
            });
          }

          // Malformed JSON is a client error, not a 500: fall into the schema
          // 400 below instead of the catch-all (which echoes error messages).
          const body: unknown = await request.json().catch(() => null);
          const validationResult = validationResultSchema.safeParse(body);
          if (!validationResult.success) {
            return new Response(
              JSON.stringify({
                error:
                  validationResult.error.errors[0]?.message ??
                  'Invalid request',
              }),
              {
                status: 400,
                headers: { 'Content-Type': 'application/json' },
              },
            );
          }

          await inngest.send({
            name: 'insights-agent/validation.completed',
            // userId comes from the Clerk session, never the request body —
            // the agent's waitForEvent condition pins on it so only the run's
            // initiating user can complete the validation.
            data: { ...validationResult.data, userId },
          });

          return new Response(JSON.stringify({ success: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        } catch (error) {
          return new Response(
            JSON.stringify({
              error:
                error instanceof Error
                  ? error.message
                  : 'Failed to report validation result',
            }),
            {
              status: 500,
              headers: { 'Content-Type': 'application/json' },
            },
          );
        }
      },
    },
  },
});
