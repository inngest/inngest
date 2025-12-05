import {
  PlainClient,
  type PlainSDKError,
} from "@team-plain/typescript-sdk/dist/index";
import { createServerFn } from "@tanstack/react-start";

// Initialize Plain client
// The API key should be set in the environment variable PLAIN_API_KEY
const plainClient = new PlainClient({
  apiKey: process.env.PLAIN_API_KEY || "",
});
type Data<T> = {
  data: T;
  error?: never;
};
type Err<U> = {
  data?: never;
  error: U;
};
type Result<T, U> = NonNullable<Data<T> | Err<U>>;

export type TicketSummary = {
  id: string;
  title: string;
  status: string;
  priority: string;
  createdAt: string;
  updatedAt: string;
};

export type TicketDetail = {
  id: string;
  title: string;
  description: string | null;
  status: string;
  priority: string;
  createdAt: string;
  updatedAt: string;
  customerName: string;
};

export const getLabelForStatus = (status: string) => {
  const statusStr = status ? String(status).toLowerCase() : "";
  switch (statusStr) {
    case "todo":
    case "snoozed":
      return "Open";
    case "done":
      return "Closed";
    default:
      return "Unknown";
  }
};

export const getTicketsByEmail = createServerFn({ method: "GET" })
  .inputValidator((data: { email: string }) => data)
  .handler(async ({ data }): Promise<TicketSummary[]> => {
    try {
      const { email } = data;

      // First, get or create the customer by email
      const customer = await plainClient.getCustomerByEmail({
        email,
      });

      if (!customer.data || customer.error) {
        console.error("Failed to get customer:", customer.error);
        return [];
      }

      // Fetch threads for this customer
      const threadsResult = await plainClient.getThreads({
        filters: {
          customerIds: [customer.data.id],
        },
        first: 10,
      });

      if (threadsResult.error || !threadsResult.data) {
        console.error("Failed to fetch threads:", threadsResult.error);
        return [];
      }

      // Map threads to ticket summaries
      const tickets: TicketSummary[] = threadsResult.data.threads.map(
        (thread: any) => ({
          id: thread.id,
          title: thread.title || "Untitled",
          status: String(thread.status || "UNKNOWN"),
          priority: String(thread.priority || "NORMAL"),
          createdAt: thread.createdAt.iso8601,
          updatedAt: thread.updatedAt.iso8601,
        }),
      );

      return tickets;
    } catch (error) {
      console.error("Error fetching tickets:", error);
      return [];
    }
  });

export const getTicketById = createServerFn({ method: "GET" })
  .inputValidator((data: { ticketId: string }) => data)
  .handler(async ({ data }): Promise<TicketDetail | null> => {
    try {
      const { ticketId } = data;

      const result = await plainClient.getThread({
        threadId: ticketId,
      });

      if (result.error || !result.data) {
        console.error("Failed to fetch thread:", result.error);
        return null;
      }

      const thread = result.data as any;

      return {
        id: thread.id,
        title: thread.title || "Untitled",
        description: thread.description || null,
        status: String(thread.status || "UNKNOWN"),
        priority: String(thread.priority || "NORMAL"),
        createdAt: thread.createdAt?.iso8601 || new Date().toISOString(),
        updatedAt: thread.updatedAt?.iso8601 || new Date().toISOString(),
        customerName: thread.customer?.email || "Unknown",
      };
    } catch (error) {
      console.error("Error fetching ticket details:", error);
      return null;
    }
  });

type TimeLineEntryEdge = {
  cursor: string;
  node: {
    id: string;
    timestamp: {
      __typename: string;
      iso8601: string;
    };
    actor:
      | {
          __typename: "UserActor";
          user: { fullName: string };
        }
      | {
          __typename: "CustomerActor";
          customer: { fullName: string };
        }
      | {
          __typename: "MachineUserActor";
          machineUser: { fullName: string };
        };
    entry: EmailEntry | CustomEntry; // CustomEntry | ChatEntry | SlackMessageEntry;
  };
};

type TimelineEntriesResponse = {
  thread: {
    customer: {
      fullName: string;
    };
    timelineEntries: { edges: TimeLineEntryEdge[] };
  };
};

type DateTime = {
  unixTimestamp: string;
  iso8601: string;
};

type UserActor = {
  userId: string;
  user: {
    // We can add more fields here if needed
    fullName: string;
  };
};

// NOTE - Check out "customerGroupMemberships" later on
type CustomerActor = {
  customerId: string;
  customer: {
    externalId: string;
    fullName: string;
  };
};
type Actor = UserActor | CustomerActor;

type Component = {
  __typename: "ComponentText" | "ComponentPlainText";
  // We normalize this field in the query
  text: string;
};

type CustomEntry = {
  __typename: "CustomEntry";
  title: string;
  components: Component[];
};

type EmailEntry = {
  __typename: "EmailEntry";
  emailId: string;
  // to: EmailParticipant!
  // from: EmailParticipant!
  // additionalRecipients: [EmailParticipant!]!
  // hiddenRecipients: [EmailParticipant!]!
  subject: string;

  textContent: string;
  markdownContent: string;
  hasMoreMarkdownContent: boolean;
  fullMarkdownContent: string;
  // authenticity: EmailAuthenticity!
  sentAt: DateTime;
  // sendStatus: EmailSendStatus
  // receivedAt: DateTime
  attachments: Attachment[];
  // category: EmailCategory!
};

type Attachment = {
  id: string;
  fileName: string;
  // fileSize: FileSize!
  fileExtension: string;
  fileMimeType: string;
  // type: AttachmentType;
  createdAt: DateTime;
  createdBy: Actor;
  updatedAt: DateTime;
  updatedBy: Actor;
};

export const getTimelineEntriesForTicket = createServerFn({ method: "GET" })
  .inputValidator((data: { ticketId: string }) => data)
  .handler(async ({ data }): Promise<TimeLineEntryEdge[] | null> => {
    try {
      const { ticketId } = data;

      const res = (await plainClient.rawRequest({
        query: `
          query GetTimelineEntries($threadId: ID!, $first: Int, $after: String, $last: Int, $before: String) {
            thread(threadId: $threadId) {
              id
              customer {
                fullName
              }
              timelineEntries(first: $first, after: $after, last: $last, before: $before) {
                edges {
                  cursor
                  node {
                    id
                    timestamp {
                      __typename
                      iso8601
                    }
                    actor {
                      __typename
                      ... on UserActor {
                        user {
                          fullName
                        }
                      }
                      ... on CustomerActor {
                        customer {
                          fullName
                        }
                      }
                      ... on MachineUserActor {
                        machineUser {
                          fullName
                        }
                      }
                    }
                    entry {
                      __typename
                      ... on CustomEntry {
                        title
                        components {
                          __typename
                          ... on ComponentText {
                            text
                          }
                          ... on ComponentPlainText {
                            text: plainText
                          }
                        }
                      }
                      ... on EmailEntry {
                        emailId
                        subject
                        markdownContent
                        sentAt {
                          iso8601
                        }
                        attachments {
                          fileName
                          fileExtension
                          fileMimeType
                          createdAt {
                            iso8601
                          }
                        }
                      }
                      ... on SlackMessageEntry {
                        slackMessageLink
                        slackWebMessageLink
                        text
                        customerId
                        attachments {
                          fileName
                          fileExtension
                          fileMimeType
                          createdAt {
                            iso8601
                          }
                        }
                        lastEditedOnSlackAt {
                          iso8601
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        `,
        variables: {
          threadId: ticketId,
          first: 20,
          after: null,
          last: null,
          before: null,
        },
      })) as unknown as Result<TimelineEntriesResponse, PlainSDKError>;

      if (res.error || !res.data) {
        console.error("Failed to fetch timeline entries:", res.error);
        return [];
      }

      const customerName = res.data.thread.customer.fullName;

      const entries = res.data.thread.timelineEntries.edges;
      // Filter out entries that are not EmailEntry or SlackMessageEntry
      console.log(JSON.stringify(entries, null, 2));
      return entries
        .filter(
          (entry) =>
            // Custom entries are created via the API
            entry.node.entry.__typename === "CustomEntry" ||
            entry.node.entry.__typename === "EmailEntry", //||
          // entry.node.entry.__typename === "SlackMessageEntry",
        )
        .sort(
          (a, b) =>
            new Date(a.node.timestamp.iso8601).getTime() -
            new Date(b.node.timestamp.iso8601).getTime(),
        )
        .map((entry, idx) => {
          // Map the first custom entry to the customer's name
          // We create a ticket via the API so it's a "machine user"
          if (idx === 0 && entry.node.entry.__typename === "CustomEntry") {
            return {
              ...entry,
              node: {
                ...entry.node,
                actor: {
                  __typename: "CustomerActor",
                  customer: { fullName: customerName },
                },
              },
            };
          }
          return entry;
        });
    } catch (error) {
      console.error("Error fetching timeline entries:", error);
      return [];
    }
  });
