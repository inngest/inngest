import { createFileRoute } from '@tanstack/react-router';
import { auth } from '@clerk/tanstack-react-start/server';
import { z } from 'zod/v3';
import { inngest } from '@/lib/inngest/client';

const feedbackRequestSchema = z.object({
  runId: z.string().min(1, 'runId is required'),
  executedOk: z.boolean().optional(),
  rowCount: z.number().optional(),
  userEdited: z.boolean().optional(),
  saved: z.boolean().optional(),
  fixWithAi: z.boolean().optional(),
});

export const Route = createFileRoute('/api/query-feedback')({
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

          const body = await request.json();
          const validationResult = feedbackRequestSchema.safeParse(body);
          if (!validationResult.success) {
            return new Response(
              JSON.stringify({
                error:
                  validationResult.error.errors[0]?.message ??
                  'Invalid request',
              }),
              { status: 400, headers: { 'Content-Type': 'application/json' } },
            );
          }

          await inngest.send({
            name: 'insights-agent/query.feedback',
            data: validationResult.data,
          });

          return new Response(JSON.stringify({ success: true }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          });
        } catch (error) {
          // Log the detail server-side; return a generic message so internal
          // error specifics (upstream URLs, client internals) don't leak.
          console.error('query-feedback handler failed', error);
          return new Response(
            JSON.stringify({ error: 'Failed to record feedback' }),
            { status: 500, headers: { 'Content-Type': 'application/json' } },
          );
        }
      },
    },
  },
});
