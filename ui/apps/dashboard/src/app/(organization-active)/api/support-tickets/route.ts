import { NextResponse } from 'next/server';
import { auth, currentUser } from '@clerk/nextjs/server';
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
} from '@/app/(organization-active)/support/ticketOptions';

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

function createComponents(
  ticket: RequestBody['ticket']
): UpsertCustomTimelineEntryInput['components'] {
  return [
    {
      componentText: {
        text: ticket.body,
      },
    },
  ];
}

export async function POST(req: Request) {
  const { userId } = auth();
  if (!userId) {
    return NextResponse.json({ error: 'Please sign in to create a ticket' }, { status: 401 });
  }

  // In production validation of the request body might be necessary.
  const body = (await req.json()) as RequestBody;

  console.log(body);

  const upsertCustomerRes = await client.upsertCustomer({
    identifier: {
      emailAddress: body.user.email,
    },
    onCreate: {
      externalId: body.user.id,
      fullName: body.user.name || body.user.email, // A name is required, so we cannot provide an empty string
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

  if (upsertCustomerRes.error) {
    console.error(JSON.stringify(upsertCustomerRes.error));
    return NextResponse.json({ error: upsertCustomerRes.error.message }, { status: 500 });
  }

  console.log(`Customer upserted ${upsertCustomerRes.data.customer.id}`);

  const thread: CreateThreadInput = {
    title: ticketTypeTitles[body.ticket.type],
    components: createComponents(body.ticket),
    customerIdentifier: {
      customerId: upsertCustomerRes.data.customer.id,
    },
    labelTypeIds: [labelTypeIDs[body.ticket.type]],
  };

  // Plain only supports priority 0-3, ignore the lowest for now
  // NOTE - We should probably split out the "technical guidance" issues as a separate type and
  // keep other issues as 0-3 priority customer issues
  const severity = body.ticket.severity ? parseInt(body.ticket.severity, 10) : null;
  if (severity && severity < 4) {
    thread.priority = severity;
  }

  const threadRes = await client.createThread(thread);

  if (threadRes.error) {
    console.error(JSON.stringify(threadRes.error));
    return NextResponse.json({ error: threadRes.error.message }, { status: 500 });
  }

  console.log(`Thread created ${threadRes.data.id}`);

  return NextResponse.json(
    {
      error: null,
      threadId: threadRes.data.id,
    },
    { status: 200 }
  );
}

export async function GET() {
  const user = await currentUser();

  if (!user) {
    return NextResponse.json({ error: 'Please sign in to view your tickets' }, { status: 401 });
  }
  const emails = user.emailAddresses.map((email) => email.emailAddress);
  let threads: ThreadPartsFragment[] = [];

  for (const email of emails) {
    const customerRes = await client.getCustomerByEmail({
      email,
    });
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
    const oneMonthAgo = getTimestampDaysAgo({ currentDate: new Date(), days: 30 });
    threads = threads.concat(
      threadsRes.data.threads.filter(
        (thread) =>
          parseInt(thread.updatedAt.unixTimestamp, 10) > Math.floor(oneMonthAgo.getTime() / 1000)
      )
    );
  }

  return NextResponse.json(
    {
      data: threads,
    },
    { status: 200 }
  );
}
