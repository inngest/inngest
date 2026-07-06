import { createFileRoute } from '@tanstack/react-router';
import { auth } from '@clerk/tanstack-react-start/server';
import { z } from 'zod/v3';
import { inngest } from '@/lib/inngest/client';

//
// The browser half of the agent's validate_query round trip: the chat UI runs
// the SQL with the user's own credentials and posts the outcome here, which
// forwards it as the event the agent loop is waiting on (step.waitForEvent).
const validationResultSchema = z.object({
  validationId: z.string().min(1),
  ok: z.boolean(),
  columns: z.array(z.string()).optional(),
  rowCount: z.number().optional(),
  diagnostics: z
    .array(
      z.object({
        code: z.string().optional(),
        message: z.string(),
      }),
    )
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

          const validationResult = validationResultSchema.safeParse(
            await request.json(),
          );
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
            data: validationResult.data,
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
