import { createFileRoute } from '@tanstack/react-router';
import { auth } from '@clerk/tanstack-react-start/server';

// NOTION_WAITLIST_API_KEY is the secret for the dedicated "sandbox waitlist"
// Notion integration; NOTION_WAITLIST_DATABASE_ID holds the Notion *data
// source* id (the id returned by the Notion connector). Both are read inside
// the handler so hot-reloading new env values takes effect without a full
// restart, and so the Authorization header is never built from `undefined`.

const NOTION_API_BASE = 'https://api.notion.com/v1';
// Data sources (parent.data_source_id) require the 2025-09-03 API or later.
const NOTION_VERSION = '2025-09-03';

// Notion caps each rich_text content segment at 2000 characters. Split long
// answers into multiple segments so nothing is lost (vs. a silent 400).
const NOTION_RICH_TEXT_LIMIT = 2000;
function toRichText(content: string) {
  const segments: { text: { content: string } }[] = [];
  for (let i = 0; i < content.length; i += NOTION_RICH_TEXT_LIMIT) {
    segments.push({
      text: { content: content.slice(i, i + NOTION_RICH_TEXT_LIMIT) },
    });
  }
  return segments.length ? segments : [{ text: { content: '' } }];
}

type Identity = {
  user: { name: string; email: string };
  organization: { name: string };
  page: string;
};

// `join` is fired when the modal opens (records intent immediately); `answers`
// updates that signup with the user's responses when they click Send. The row
// is always located server-side via the authenticated Clerk user id — the
// client never supplies a Notion page id, which avoids an IDOR where a crafted
// id could patch someone else's row.
export type RequestBody = Partial<Identity> & {
  action: 'join' | 'answers';
  workflow?: string;
  canContact?: boolean;
};

function notionHeaders(apiKey: string) {
  return {
    Authorization: `Bearer ${apiKey}`,
    'Notion-Version': NOTION_VERSION,
    'Content-Type': 'application/json',
  };
}

// Identity properties match the "Sandboxes [Waitlist]" Notion data source:
// Name (title), User (rich_text), Organisation (rich_text), Page (url),
// Clerk User ID (rich_text — the server-trusted owner used to locate the row).
function identityProperties(identity: Identity, userId: string) {
  return {
    Name: { title: [{ text: { content: identity.user.name.slice(0, 60) } }] },
    User: {
      rich_text: [
        { text: { content: `${identity.user.name} <${identity.user.email}>` } },
      ],
    },
    Organisation: {
      rich_text: [{ text: { content: identity.organization.name } }],
    },
    Page: { url: identity.page },
    'Clerk User ID': { rich_text: [{ text: { content: userId } }] },
  };
}

// Answer properties: Workflow (rich_text), Can contact (checkbox).
function answerProperties(workflow: string, canContact: boolean) {
  return {
    Workflow: { rich_text: toRichText(workflow) },
    'Can contact': { checkbox: canContact },
  };
}

function isIdentity(body: RequestBody): body is RequestBody & Identity {
  return Boolean(body.user && body.organization && body.page);
}

function json(body: unknown, status: number) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

// Locate the current user's own signup row by the server-trusted Clerk user id.
// Returns the most recent match, or null if none exists / the query fails.
async function findOwnRowId(
  apiKey: string,
  dataSourceId: string,
  userId: string,
): Promise<string | null> {
  const res = await fetch(
    `${NOTION_API_BASE}/data_sources/${dataSourceId}/query`,
    {
      method: 'POST',
      headers: notionHeaders(apiKey),
      body: JSON.stringify({
        filter: { property: 'Clerk User ID', rich_text: { equals: userId } },
        sorts: [{ timestamp: 'created_time', direction: 'descending' }],
        page_size: 1,
      }),
    },
  );
  if (!res.ok) {
    console.error(
      'error querying waitlist rows via Notion API',
      await res.text().catch(() => null),
    );
    return null;
  }
  const data = (await res.json().catch(() => null)) as {
    results?: { id: string }[];
  } | null;
  return data?.results?.[0]?.id ?? null;
}

export const Route = createFileRoute('/api/waitlist')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        const apiKey = process.env.NOTION_WAITLIST_API_KEY;
        const dataSourceId = process.env.NOTION_WAITLIST_DATABASE_ID;
        if (!apiKey || !dataSourceId) {
          console.error(
            'waitlist endpoint is missing NOTION_WAITLIST_API_KEY or NOTION_WAITLIST_DATABASE_ID',
          );
          return json({ error: 'Waitlist is not configured' }, 501);
        }

        const { userId } = await auth();
        if (!userId) {
          return json({ error: 'Please sign in to join the waitlist' }, 401);
        }

        const body = (await request.json()) as RequestBody;
        const headers = notionHeaders(apiKey);
        const submitting = body.action === 'answers';

        // Answers: patch this user's own row (located server-side). If they
        // have no row yet (join create failed / never fired), fall through to
        // create one below with the answers included.
        if (submitting) {
          const ownRowId = await findOwnRowId(apiKey, dataSourceId, userId);
          if (ownRowId) {
            const res = await fetch(`${NOTION_API_BASE}/pages/${ownRowId}`, {
              method: 'PATCH',
              headers,
              body: JSON.stringify({
                properties: {
                  ...answerProperties(
                    body.workflow ?? '',
                    body.canContact === true,
                  ),
                  Status: { select: { name: 'Submitted' } },
                },
              }),
            });
            if (!res.ok) {
              console.error(
                'error updating waitlist row via Notion API',
                await res.text().catch(() => null),
              );
              return json({ error: 'Failed to submit answers' }, 500);
            }
            return json({ error: null }, 200);
          }
        } else {
          // Join: keep signups one-per-user — if a row already exists, the
          // click is already recorded, so don't create a duplicate.
          const ownRowId = await findOwnRowId(apiKey, dataSourceId, userId);
          if (ownRowId) {
            return json({ error: null }, 200);
          }
        }

        // Create a new row: the initial `join`, or an `answers` submission with
        // no existing row (fallback).
        if (!isIdentity(body)) {
          return json({ error: 'Missing user or organization' }, 400);
        }

        const res = await fetch(`${NOTION_API_BASE}/pages`, {
          method: 'POST',
          headers,
          body: JSON.stringify({
            parent: { type: 'data_source_id', data_source_id: dataSourceId },
            properties: {
              ...identityProperties(body, userId),
              ...(submitting
                ? answerProperties(
                    body.workflow ?? '',
                    body.canContact === true,
                  )
                : {}),
              Status: { select: { name: submitting ? 'Submitted' : 'Joined' } },
            },
          }),
        });

        if (!res.ok) {
          console.error(
            'error creating waitlist row via Notion API',
            await res.text().catch(() => null),
          );
          return json({ error: 'Failed to join the waitlist' }, 500);
        }

        return json({ error: null }, 200);
      },
    },
  },
});
