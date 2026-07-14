import { createFileRoute } from '@tanstack/react-router';
import { auth } from '@clerk/tanstack-react-start/server';

// Read lazily; validated inside the handler so a missing config only fails the
// waitlist endpoint rather than crashing the whole app at module-load time.
// NOTION_WAITLIST_API_KEY is the secret for the dedicated "sandbox waitlist"
// Notion integration; NOTION_WAITLIST_DATABASE_ID holds the Notion *data
// source* id (the id returned by the Notion connector), addressed below via
// the data-source API.
const apiKey = process.env.NOTION_WAITLIST_API_KEY;
const dataSourceId = process.env.NOTION_WAITLIST_DATABASE_ID;

const NOTION_PAGES_URL = 'https://api.notion.com/v1/pages';
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
// updates that row when the user clicks Send. `answers` falls back to creating
// a row if the original `join` create failed (no pageId).
export type RequestBody = Partial<Identity> & {
  action: 'join' | 'answers';
  pageId?: string;
  workflow?: string;
  canContact?: boolean;
};

const notionHeaders = {
  Authorization: `Bearer ${apiKey}`,
  'Notion-Version': NOTION_VERSION,
  'Content-Type': 'application/json',
};

// Identity properties match the "Sandbox waitlist" Notion data source:
// Name (title), User (rich_text), Organisation (rich_text), Page (url).
function identityProperties(identity: Identity) {
  return {
    Name: {
      title: [{ text: { content: identity.user.name.slice(0, 60) } }],
    },
    User: {
      rich_text: [
        { text: { content: `${identity.user.name} <${identity.user.email}>` } },
      ],
    },
    Organisation: {
      rich_text: [{ text: { content: identity.organization.name } }],
    },
    Page: { url: identity.page },
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

export const Route = createFileRoute('/api/waitlist')({
  server: {
    handlers: {
      POST: async ({ request }) => {
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

        // Update an existing signup row with the user's answers.
        if (body.action === 'answers' && body.pageId) {
          const res = await fetch(`${NOTION_PAGES_URL}/${body.pageId}`, {
            method: 'PATCH',
            headers: notionHeaders,
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

        // Otherwise create a new row: either the initial `join`, or an `answers`
        // submission whose `join` create never landed (fallback).
        if (!isIdentity(body)) {
          return json({ error: 'Missing user or organization' }, 400);
        }

        const submitting = body.action === 'answers';
        const res = await fetch(NOTION_PAGES_URL, {
          method: 'POST',
          headers: notionHeaders,
          body: JSON.stringify({
            parent: { type: 'data_source_id', data_source_id: dataSourceId },
            properties: {
              ...identityProperties(body),
              ...(submitting
                ? answerProperties(
                    body.workflow ?? '',
                    body.canContact === true,
                  )
                : {}),
              Status: {
                select: { name: submitting ? 'Submitted' : 'Joined' },
              },
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

        const created = (await res.json().catch(() => null)) as {
          id?: string;
        } | null;
        return json({ error: null, pageId: created?.id ?? null }, 200);
      },
    },
  },
});
