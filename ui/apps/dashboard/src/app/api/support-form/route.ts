import { NextResponse } from 'next/server';
import {
  PlainClient,
  type CreateIssueInput,
  type UpsertCustomTimelineEntryInput,
} from '@team-plain/typescript-sdk';

import { ticketTypeTitles, type BugSeverity, type TicketType } from '../../support/ticketOptions';

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

const issueTypeIDs: { [K in Exclude<TicketType, null>]: string } = {
  bug: process.env.PLAIN_ISSUE_TYPE_ID_BUG || '',
  demo: process.env.PLAIN_ISSUE_TYPE_ID_DEMO || '',
  billing: process.env.PLAIN_ISSUE_TYPE_ID_BILLING || '',
  feature: process.env.PLAIN_ISSUE_TYPE_ID_FEATURE_REQUEST || '',
  security: process.env.PLAIN_ISSUE_TYPE_ID_SECURITY || '',
  question: process.env.PLAIN_ISSUE_TYPE_ID_QUESTION || '',
} as const;

export async function POST(req: Request) {
  // In production validation of the request body might be necessary.
  const body = (await req.json()) as RequestBody;

  console.log(body);

  const upsertCustomerRes = await client.upsertCustomer({
    identifier: {
      // Use only one identifier in the system - we'll use email for now as the user might have emailed before using the form
      // externalId: body.user.id,
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

  const upsertTimelineEntryRes = await client.upsertCustomTimelineEntry({
    customerId: upsertCustomerRes.data.customer.id,
    title: ticketTypeTitles[body.ticket.type],
    components: createComponents(body.ticket),
    changeCustomerStatusToActive: true,
  });

  if (upsertTimelineEntryRes.error) {
    console.error(upsertTimelineEntryRes.error);
    return NextResponse.json({ error: upsertTimelineEntryRes.error.message }, { status: 500 });
  }

  console.log(`Custom timeline entry upserted ${upsertTimelineEntryRes.data.timelineEntry.id}.`);

  const issue: CreateIssueInput = {
    customerId: upsertCustomerRes.data.customer.id,
    issueTypeId: issueTypeIDs[body.ticket.type],
  };

  // Plain only supports priority 0-3, ignore the lowest for now
  // NOTE - We should probably split out the "technical guidance" issues as a separate type and
  // keep other issues as 0-3 priority customer issues
  const severity = body.ticket.severity ? parseInt(body.ticket.severity, 10) : null;
  if (severity && severity < 4) {
    issue.priorityValue = severity;
  }

  const createIssueRes = await client.createIssue(issue);

  if (createIssueRes.error) {
    console.error(createIssueRes.error);
    return NextResponse.json({ error: createIssueRes.error.message }, { status: 500 });
  }

  console.log(`Issue created ${createIssueRes.data.id}`);

  return NextResponse.json({ error: null }, { status: 200 });
}
