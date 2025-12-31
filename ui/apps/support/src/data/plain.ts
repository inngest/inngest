import {
  PlainClient,
  ThreadPartsFragment,
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

export type TicketChannel = "EMAIL" | "SLACK" | "API" | "DISCORD";

export type TicketSummary = {
  id: string;
  ref: string;
  title: string;
  status: string;
  priority: number;
  createdAt: string;
  updatedAt: string;
  channel?: TicketChannel;
  previewText?: string;
};

export type TicketDetail = {
  id: string;
  ref: string;
  title: string;
  description: string | null;
  status: string;
  priority: number;
  channel?: TicketChannel;
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
    // TODO - Use Clerk auth here to get the customer email, and the metadata with their external id
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

      const res = (await plainClient.rawRequest({
        query: `
          query GetThreads($filters: ThreadsFilter, $sortBy: ThreadsSort, $first: Int, $after: String, $last: Int, $before: String) {
            threads(filters: $filters, sortBy: $sortBy, first: $first, after: $after, last: $last, before: $before) {
              pageInfo {
                hasNextPage
                hasPreviousPage
                startCursor
                endCursor
              }
              totalCount
              edges {
                cursor
                node {
                  id
                  ref
                  title
                  status
                  priority
                  previewText
                  createdAt {
                    unixTimestamp
                    iso8601
                  }
                  statusChangedAt {
                    unixTimestamp
                    iso8601
                  }
                  updatedAt {
                    unixTimestamp
                    iso8601
                  }
                  customer {
                    fullName
                  }
                  channel
                }
              }
            }
          }
        `,
        variables: {
          filters: {
            customerIds: [customer.data.id],
          },
          first: 10,
          // after: null,
          // last: null,
          // before: null,
        },
      })) as unknown as Result<ThreadsQueryResult, PlainSDKError>;

      if (res.error || !res.data) {
        console.error("Failed to fetch threads:", res.error);
        return [];
      }

      // Map threads to ticket summaries
      const tickets: TicketSummary[] = res.data.threads.edges.map(
        (
          edge: ThreadsQueryResult["threads"]["edges"][number],
          index: number,
        ) => {
          const thread = edge.node;

          return {
            id: thread.id,
            ref: thread.ref || "",
            title: thread.title || "Untitled",
            status: String(thread.status || "UNKNOWN"),
            priority: thread.priority,
            createdAt: thread.createdAt.iso8601,
            updatedAt: thread.updatedAt.iso8601,
            channel: thread.channel as TicketChannel,
            previewText: thread.previewText || "",
          };
        },
      );

      return tickets;
    } catch (error) {
      console.error("Error fetching tickets:", error);
      return [];
    }
  });

type ThreadsQueryResult = {
  threads: {
    edges: {
      node: ThreadPartsFragment & {
        ref: string;
        previewText: string;
        channel: string;
      };
    }[];
    pageInfo: {
      hasNextPage: boolean;
      hasPreviousPage: boolean;
      startCursor: string;
      endCursor: string;
    };
    totalCount: number;
  };
};

export const getTicketById = createServerFn({ method: "GET" })
  .inputValidator((data: { ticketId: string }) => data)
  .handler(async ({ data }): Promise<TicketDetail | null> => {
    try {
      const { ticketId } = data;

      const res = (await plainClient.rawRequest({
        query: `
          query GetThread($threadId: ID!) {
            thread(threadId: $threadId) {
              id
              ref
              title
              description
              status
              priority
              channel
              createdAt {
                iso8601
              }
              updatedAt {
                iso8601
              }
              customer {
                fullName
                email {
                  email
                }
              }
            }
          }
        `,
        variables: {
          threadId: ticketId,
        },
      })) as unknown as Result<ThreadQueryResult, PlainSDKError>;

      if (res.error || !res.data) {
        console.error("Failed to fetch thread:", res.error);
        return null;
      }

      const thread = res.data.thread;

      return {
        id: thread.id,
        ref: thread.ref || "",
        title: thread.title || "Untitled",
        description: thread.description || null,
        status: String(thread.status || "UNKNOWN"),
        priority: thread.priority,
        channel: thread.channel as TicketChannel | undefined,
        createdAt: thread.createdAt?.iso8601 || new Date().toISOString(),
        updatedAt: thread.updatedAt?.iso8601 || new Date().toISOString(),
        customerName:
          thread.customer.fullName ||
          thread.customer.email?.email ||
          "Inngest user",
      };
    } catch (error) {
      console.error("Error fetching ticket details:", error);
      return null;
    }
  });

type ThreadQueryResult = {
  thread: ThreadPartsFragment & {
    ref: string;
    previewText: string;
    channel: string;
    customer: {
      fullName: string;
      email: {
        email: string;
      };
    };
  };
};

export type TimeLineEntryEdge = {
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
          customer: {
            fullName: string;
            avatarUrl: string;
            email: { email: string };
          };
        }
      | {
          __typename: "MachineUserActor";
          machineUser: { fullName: string };
        };
    entry: EmailEntry | CustomEntry | SlackMessageEntry | SlackReplyEntry;
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

export type Attachment = {
  id: string;
  fileName: string;
  fileSize: {
    megaBytes: number;
  };
  fileExtension: string;
  fileMimeType: string;
  type: "EMAIL" | "SLACK"; // Note - there are other types
  createdAt: DateTime;
  createdBy: Actor;
  updatedAt: DateTime;
  updatedBy: Actor;
};

type SlackMessageEntry = {
  __typename: "SlackMessageEntry";
  slackMessageLink: string;
  slackWebMessageLink: string;
  text: string;
  customerId: string;
  attachments: Attachment[];
  lastEditedOnSlackAt: DateTime;
};

type SlackReplyEntry = {
  __typename: "SlackReplyEntry";
  slackMessageLink: string;
  slackWebMessageLink: string;
  text: string;
  customerId: string;
  attachments: Attachment[];
  lastEditedOnSlackAt: DateTime;
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
                          avatarUrl
                          email {
                            email
                          }
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
                          id
                          fileName
                        }
                      }
                      ... on SlackMessageEntry {
                        slackMessageLink
                        slackWebMessageLink
                        text
                        customerId
                        attachments {
                          id
                          fileName
                        }
                        lastEditedOnSlackAt {
                          iso8601
                          unixTimestamp
                        }
                      }
                      ... on SlackReplyEntry {
                        slackMessageLink
                        slackWebMessageLink
                        text
                        customerId
                        attachments {
                          id
                          fileName
                        }
                        lastEditedOnSlackAt {
                          iso8601
                          unixTimestamp
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
      return entries
        .filter(
          (entry) =>
            // Custom entries are created via the API
            entry.node.entry.__typename === "CustomEntry" ||
            entry.node.entry.__typename === "EmailEntry" ||
            entry.node.entry.__typename === "SlackMessageEntry" ||
            entry.node.entry.__typename === "SlackReplyEntry",
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

export const getAttachmentDownloadUrl = createServerFn({ method: "GET" })
  .inputValidator((data: { attachmentId: string }) => data)
  .handler(async ({ data }): Promise<AttachmentDownloadUrl | null> => {
    try {
      const { attachmentId } = data;

      const res = (await plainClient.rawRequest({
        query: `
          mutation CreateAttachmentDownloadUrl($attachmentId: ID!) {
            createAttachmentDownloadUrl(input: { attachmentId: $attachmentId }) {
              attachmentDownloadUrl {
                attachment {
                  id
                  fileName
                  fileExtension
                  fileMimeType
                  createdAt {
                    iso8601
                  }
                }
                downloadUrl
                expiresAt {
                  iso8601
                }
              }
            }
          }
        `,
        variables: {
          attachmentId,
        },
      })) as unknown as Result<
        CreateAttachmentDownloadUrlOutput,
        PlainSDKError
      >;

      if (res.error || !res.data) {
        console.error("Failed to fetch attachment download url:", res.error);
        return null;
      }

      console.log(
        "Attachment download url loader: ",
        res.data.createAttachmentDownloadUrl.attachmentDownloadUrl,
      );

      return res.data.createAttachmentDownloadUrl.attachmentDownloadUrl;
    } catch (error) {
      console.error("Error fetching attachment download url:", error);
      return null;
    }
  });

type CreateAttachmentDownloadUrlOutput = {
  createAttachmentDownloadUrl: {
    attachmentDownloadUrl: AttachmentDownloadUrl;
  };
};

type AttachmentDownloadUrl = {
  attachment: Attachment;
  downloadUrl: string;
  expiresAt: DateTime;
};
