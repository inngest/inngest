import { createFileRoute } from '@tanstack/react-router';
import { auth } from '@clerk/tanstack-react-start/server';

const apiKey = process.env.NOTION_API_KEY;
const databaseId = process.env.NOTION_FEEDBACK_DATABASE_ID;

if (!apiKey) {
  throw new Error('NOTION_API_KEY environment variable is not set');
}

if (!databaseId) {
  throw new Error(
    'NOTION_FEEDBACK_DATABASE_ID environment variable is not set',
  );
}

// Non-secret, fixed per Clerk account. Points at the production instance so
// feedback rows deep-link to the real user/org.
const CLERK_DASHBOARD_BASE =
  'https://dashboard.clerk.com/apps/app_2QsTVg17CfOaTyorzo4ojtU3qd0/instances/ins_2ULeE7m3FH0dgTOa6mK56AtYtXZ';

const NOTION_API_URL = 'https://api.notion.com/v1/pages';
const NOTION_VERSION = '2022-06-28';

export type RequestBody = {
  user: { name: string; email: string; clerkId: string };
  organization: { name: string; clerkId: string };
  page: string;
  feedback: string;
};

export const Route = createFileRoute('/api/feedback')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        const { userId } = await auth();
        if (!userId) {
          return new Response(
            JSON.stringify({ error: 'Please sign in to send feedback' }),
            {
              status: 401,
              headers: { 'Content-Type': 'application/json' },
            },
          );
        }

        const body = (await request.json()) as RequestBody;

        const feedback = body.feedback?.trim();
        if (!feedback) {
          return new Response(
            JSON.stringify({ error: 'Feedback cannot be empty' }),
            {
              status: 400,
              headers: { 'Content-Type': 'application/json' },
            },
          );
        }

        const userUrl = `${CLERK_DASHBOARD_BASE}/users/${body.user.clerkId}`;
        const orgUrl = `${CLERK_DASHBOARD_BASE}/organizations/${body.organization.clerkId}`;

        // Property names/types match the "Raw User Feedback" Notion database:
        // User (rich_text), User Clerk (url), Organisation (rich_text),
        // Organisation Clerk (url), Page (url), Name (title).
        const notionRes = await fetch(NOTION_API_URL, {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${apiKey}`,
            'Notion-Version': NOTION_VERSION,
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            parent: { database_id: databaseId },
            properties: {
              Name: {
                title: [{ text: { content: feedback.slice(0, 60) } }],
              },
              User: {
                rich_text: [
                  {
                    text: { content: `${body.user.name} <${body.user.email}>` },
                  },
                ],
              },
              'User Clerk': { url: userUrl },
              Organisation: {
                rich_text: [{ text: { content: body.organization.name } }],
              },
              'Organisation Clerk': { url: orgUrl },
              Page: { url: body.page },
            },
            children: [
              {
                object: 'block',
                type: 'paragraph',
                paragraph: {
                  rich_text: [{ text: { content: feedback } }],
                },
              },
            ],
          }),
        });

        if (!notionRes.ok) {
          console.error(
            'error creating feedback via Notion API',
            await notionRes.text().catch(() => null),
          );
          return new Response(
            JSON.stringify({ error: 'Failed to submit feedback' }),
            {
              status: 500,
              headers: { 'Content-Type': 'application/json' },
            },
          );
        }

        return new Response(JSON.stringify({ error: null }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      },
    },
  },
});
