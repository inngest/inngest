import { createFileRoute } from '@tanstack/react-router';
import { auth, clerkClient } from '@clerk/tanstack-react-start/server';
import { getTimestampDaysAgo } from '@inngest/components/utils/date';
import {
  PlainClient,
  type CreateThreadInput,
  type ThreadPartsFragment,
  type UpsertCustomTimelineEntryInput,
} from '@team-plain/typescript-sdk';

import {
  labelTypeIDs,
  ticketTypeTitles,
  type BugSeverity,
  type TicketType,
} from '@/components/Support/ticketOptions';

const apiKey = process.env.PLAIN_API_KEY;

if (!apiKey) {
  throw new Error('PLAIN_API_KEY environment variable is not set');
}

const client = new PlainClient({
  apiKey,
});

export type RequestBody = {
  user: {
    id: string;
    email: string;
    name?: string;
    accountId: string;
  };
  ticket: {
    type: Exclude<TicketType, null>;
    body: string;
    severity?: BugSeverity;
  };
};

const createComponents = (
  ticket: RequestBody['ticket'],
): UpsertCustomTimelineEntryInput['components'] => {
  return [
    {
      componentText: {
        text: ticket.body,
      },
    },
  ];
};

//
// brute force find or create a customer in plain to work around
// upsert issues in the API when users change or re-use email addresses
// across orgs
const getCustomerId = async (email: string, name?: string) => {
  const existing = await client.getCustomerByEmail({
    email,
  });

  if (existing.data?.id) {
    return existing.data.id;
  }

  const upserted = await client.upsertCustomer({
    identifier: {
      emailAddress: email,
    },
    onCreate: {
      fullName: name || email,
      email: {
        email: email,
        isVerified: true,
      },
    },
    onUpdate: {
      fullName: { value: name || email },
      email: {
        email: email,
        isVerified: true,
      },
    },
  });

  return upserted.data?.customer.id;
};

export const Route = createFileRoute('/api/support-tickets')({
  server: {
    handlers: {
      POST: async ({ request }) => {
        const { userId } = await auth();
        if (!userId) {
          return new Response(
            JSON.stringify({ error: 'Please sign in to create a ticket' }),
            {
              status: 401,
              headers: { 'Content-Type': 'application/json' },
            },
          );
        }

        const body = (await request.json()) as RequestBody;

        const upsertedCustomer = await client.upsertCustomer({
          identifier: {
            emailAddress: body.user.email,
          },
          onCreate: {
            externalId: body.user.id,
            fullName: body.user.name || body.user.email,
            email: {
              email: body.user.email,
              isVerified: true,
            },
          },
          onUpdate: {
            externalId: { value: body.user.id },
            fullName: body.user.name ? { value: body.user.name } : undefined,
            email: {
              email: body.user.email,
              isVerified: true,
            },
          },
        });

        const customerId = upsertedCustomer.data?.customer.id
          ? upsertedCustomer.data.customer.id
          : await getCustomerId(body.user.email, body.user.name);

        const thread: CreateThreadInput = {
          title: ticketTypeTitles[body.ticket.type],
          components: createComponents(body.ticket),
          customerIdentifier: {
            customerId,
          },
          labelTypeIds: [labelTypeIDs[body.ticket.type]],
        };

        //
        // Plain only supports priority 0-3, ignore the lowest for now
        // NOTE - We should probably split out the "technical guidance" issues as a separate type and
        // keep other issues as 0-3 priority customer issues
        const severity = body.ticket.severity
          ? parseInt(body.ticket.severity, 10)
          : null;
        if (severity && severity < 4) {
          thread.priority = severity;
        }

        const threadRes = await client.createThread(thread);

        if (threadRes.error) {
          console.error(
            'error creating ticket via support API',
            JSON.stringify(threadRes.error),
          );
          return new Response(
            JSON.stringify({ error: threadRes.error.message }),
            {
              status: 500,
              headers: { 'Content-Type': 'application/json' },
            },
          );
        }

        return new Response(
          JSON.stringify({
            error: null,
            threadId: threadRes.data.id,
          }),
          {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          },
        );
      },

      GET: async () => {
        const { userId } = await auth();

        if (!userId) {
          return new Response(
            JSON.stringify({ error: 'Please sign in to view your tickets' }),
            {
              status: 401,
              headers: { 'Content-Type': 'application/json' },
            },
          );
        }

        const user = await clerkClient().users.getUser(userId);
        if (!user) {
          return new Response(JSON.stringify({ error: 'User not found' }), {
            status: 404,
            headers: { 'Content-Type': 'application/json' },
          });
        }

        const emails = user.emailAddresses.map((email) => email.emailAddress);
        let threads: ThreadPartsFragment[] = [];

        for (const email of emails) {
          const customerRes = await client.getCustomerByEmail({
            email,
          });
          //
          // Ignore if the customer doesn't yet exist in Plain
          if (customerRes.error || !customerRes.data?.id) {
            continue;
          }

          const threadsRes = await client.getThreads({
            filters: {
              customerIds: [customerRes.data.id],
            },
            first: 10,
          });
          if (threadsRes.error) {
            continue;
          }
          const oneMonthAgo = getTimestampDaysAgo({
            currentDate: new Date(),
            days: 30,
          });
          threads = threads.concat(
            threadsRes.data.threads.filter(
              (thread) =>
                parseInt(thread.updatedAt.unixTimestamp, 10) >
                Math.floor(oneMonthAgo.getTime() / 1000),
            ),
          );
        }

        return new Response(
          JSON.stringify({
            data: threads,
          }),
          {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          },
        );
      },
    },
  },
});
