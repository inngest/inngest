import {
  PlainClient,
  AttachmentType,
} from "@team-plain/typescript-sdk/dist/index";
import { createServerFn } from "@tanstack/react-start";
import type {
  PlainSDKError,
  ThreadPartsFragment,
} from "@team-plain/typescript-sdk/dist/index";
import { authMiddleware } from "@/data/clerk";

// Initialize Plain client
// The API key should be set in the environment variable PLAIN_API_KEY
const plainClient = new PlainClient({
  apiKey: process.env.PLAIN_API_KEY || "",
});
type Data<T> = {
  data: T;
  error?: never;
};
type Err<TError> = {
  data?: never;
  error: TError;
};
type Result<T, TError> = NonNullable<Data<T> | Err<TError>>;

function hasMissingLabelTypeError(error: unknown): boolean {
  if (!error || typeof error !== "object") return false;

  const maybeErrorDetails = (
    error as {
      errorDetails?: {
        fields?: Array<{ field?: string; type?: string }>;
      };
    }
  ).errorDetails;

  const fields = maybeErrorDetails?.fields;
  if (!Array.isArray(fields)) return false;

  return fields.some(
    (field) => field.field === "labelTypeIds" && field.type === "NOT_FOUND",
  );
}

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
  slackChannelLink?: string;
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

export const TICKET_STATUS_ALL = "all" as const;
export const TICKET_STATUS_OPEN = "open" as const;
export const TICKET_STATUS_CLOSED = "closed" as const;

export type TicketStatusFilter =
  | typeof TICKET_STATUS_OPEN
  | typeof TICKET_STATUS_CLOSED
  | typeof TICKET_STATUS_ALL;

export const getTicketsByEmail = createServerFn({ method: "GET" })
  .middleware([authMiddleware])
  .inputValidator((data: { status?: TicketStatusFilter }) => data)
  .handler(async ({ data, context }): Promise<Array<TicketSummary>> => {
    try {
      const email = context.userEmail;
      const { status } = data;

      // First, get or create the customer by email
      const customer = await plainClient.getCustomerByEmail({
        email,
      });

      if (customer.error) {
        console.error("Failed to get customer:", customer.error);
        return [];
      }
      if (!customer.data) {
        console.error("Customer not found");
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
            ...(status === "open"
              ? { statuses: ["TODO", "SNOOZED"] }
              : status === "closed"
              ? { statuses: ["DONE"] }
              : {}),
          },
          first: 10,
          // after: null,
          // last: null,
          // before: null,
        },
      })) as unknown as Result<ThreadsQueryResult, PlainSDKError>;

      if (res.error) {
        console.error("Failed to fetch threads:", res.error);
        return [];
      }

      // Map threads to ticket summaries
      const tickets: Array<TicketSummary> = res.data.threads.edges.map(
        (edge: ThreadsQueryResult["threads"]["edges"][number]) => {
          const thread = edge.node;
          return {
            id: thread.id,
            ref: thread.ref || "",
            title: thread.title || "Untitled",
            // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
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
    edges: Array<{
      node: ThreadPartsFragment & {
        ref: string;
        previewText: string;
        channel: string;
      };
    }>;
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
  .middleware([authMiddleware])
  .inputValidator((data: { ticketId: string }) => data)
  .handler(async ({ data, context }): Promise<TicketDetail | null> => {
    try {
      const { ticketId } = data;
      const userEmail = context.userEmail;

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

      if (res.error) {
        console.error("Failed to fetch thread:", res.error);
        return null;
      }

      const thread = res.data.thread;

      // Verify the authenticated user owns this ticket
      if (
        thread.customer.email.email.toLowerCase() !== userEmail.toLowerCase()
      ) {
        return null;
      }

      return {
        id: thread.id,
        ref: thread.ref || "",
        title: thread.title || "Untitled",
        description: thread.description || null,
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        status: String(thread.status || "UNKNOWN"),
        priority: thread.priority,
        channel: thread.channel as TicketChannel | undefined,
        createdAt: thread.createdAt.iso8601,
        updatedAt: thread.updatedAt.iso8601,
        customerName:
          thread.customer.fullName ||
          // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
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
    entry:
      | EmailEntry
      | CustomEntry
      | SlackMessageEntry
      | SlackReplyEntry
      | ChatEntry;
  };
};

type TimelineEntriesResponse = {
  thread: {
    customer: {
      fullName: string;
      email: {
        email: string;
      };
    };
    timelineEntries: { edges: Array<TimeLineEntryEdge> };
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
  components: Array<Component>;
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
  attachments: Array<Attachment>;
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
  attachments: Array<Attachment>;
  lastEditedOnSlackAt: DateTime;
};

type SlackReplyEntry = {
  __typename: "SlackReplyEntry";
  slackMessageLink: string;

  slackWebMessageLink: string;
  text: string;
  customerId: string;
  attachments: Array<Attachment>;
  lastEditedOnSlackAt: DateTime;
};

type ChatEntry = {
  __typename: "ChatEntry";
  chatId: string;
  /** Aliased as `chatText` in the query to avoid GraphQL type conflict with Slack entries */
  chatText: string;
  attachments: Array<Attachment>;
};
export const getTimelineEntriesForTicket = createServerFn({ method: "GET" })
  .middleware([authMiddleware])
  .inputValidator((data: { ticketId: string }) => data)
  .handler(
    async ({ data, context }): Promise<Array<TimeLineEntryEdge> | null> => {
      try {
        const { ticketId } = data;
        const userEmail = context.userEmail;

        const res = (await plainClient.rawRequest({
          query: `
          query GetTimelineEntries($threadId: ID!, $first: Int, $after: String, $last: Int, $before: String) {
            thread(threadId: $threadId) {
              id
              customer {
                fullName
                email {
                  email
                }
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
                      ... on ChatEntry {
                        chatId
                        chatText: text
                        attachments {
                          id
                          fileName
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

        if (res.error) {
          console.error("Failed to fetch timeline entries:", res.error);
          return [];
        }

        // Verify the authenticated user owns this ticket
        const customerEmail = res.data.thread.customer.email.email;
        if (customerEmail.toLowerCase() !== userEmail.toLowerCase()) {
          return null;
        }

        const customerName = res.data.thread.customer.fullName;
        const entries = res.data.thread.timelineEntries.edges;
        return entries
          .filter((entry) => {
            // Custom entries are created via the API
            const typename = entry.node.entry.__typename;
            return (
              typename === "CustomEntry" ||
              typename === "EmailEntry" ||
              typename === "SlackMessageEntry" ||
              // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
              typename === "SlackReplyEntry" ||
              typename === "ChatEntry"
            );
          })
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
                    customer: {
                      fullName: customerName,
                      avatarUrl: "",
                      email: { email: "" },
                    },
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
    },
  );

export const getAttachmentDownloadUrl = createServerFn({ method: "GET" })
  .middleware([authMiddleware])
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

      if (res.error) {
        console.error("Failed to fetch attachment download url:", res.error);
        return null;
      }

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

// Create ticket types
export type CreateTicketInput = {
  user: {
    id: string;
    /** @deprecated No longer used - email is derived from authenticated session */
    email?: string;
    name?: string;
  };
  ticket: {
    type: string;
    body: string;
    severity?: string;
    attachmentIds?: string[];
  };
};

export type CreateTicketResult = {
  success: boolean;
  threadId?: string;
  error?: string;
};

export const createTicket = createServerFn({ method: "POST" })
  .middleware([authMiddleware])
  .inputValidator((data: CreateTicketInput) => data)
  .handler(async ({ data, context }): Promise<CreateTicketResult> => {
    try {
      const authEmail = context.userEmail;
      const { user, ticket } = data;

      // Import label type IDs dynamically to avoid issues with env vars
      const labelTypeIDs: Record<string, string> = {
        bug: process.env.PLAIN_LABEL_TYPE_ID_BUG || "",
        demo: process.env.PLAIN_LABEL_TYPE_ID_DEMO || "",
        billing: process.env.PLAIN_LABEL_TYPE_ID_BILLING || "",
        feature: process.env.PLAIN_LABEL_TYPE_ID_FEATURE_REQUEST || "",
        security: process.env.PLAIN_LABEL_TYPE_ID_SECURITY || "",
        question: process.env.PLAIN_LABEL_TYPE_ID_QUESTION || "",
      };

      const ticketTypeTitles: Record<string, string> = {
        bug: "Bug report",
        demo: "Demo request",
        billing: "Billing issue",
        feature: "Feature request",
        security: "Security report",
        question: "General question",
      };

      // Get or create customer using the authenticated email
      const existingCustomer = await plainClient.getCustomerByEmail({
        email: authEmail,
      });

      let customerId = existingCustomer.data?.id;

      if (!customerId) {
        const upsertedCustomer = await plainClient.upsertCustomer({
          identifier: {
            emailAddress: authEmail,
          },
          onCreate: {
            externalId: user.id,
            fullName: user.name || authEmail,
            email: {
              email: authEmail,
              isVerified: true,
            },
          },
          onUpdate: {
            externalId: { value: user.id },
            fullName: user.name ? { value: user.name } : undefined,
            email: {
              email: authEmail,
              isVerified: true,
            },
          },
        });
        customerId = upsertedCustomer.data?.customer.id;
      }

      if (!customerId) {
        return {
          success: false,
          error: "Failed to create or find customer",
        };
      }

      // Create thread
      const threadInput: {
        title: string;
        components: Array<{ componentText: { text: string } }>;
        customerIdentifier: { customerId: string };
        labelTypeIds?: Array<string>;
        priority?: number;
        attachmentIds?: Array<string>;
      } = {
        title: ticketTypeTitles[ticket.type] || "Support request",
        components: [
          {
            componentText: {
              text: ticket.body,
            },
          },
        ],
        customerIdentifier: {
          customerId,
        },
      };

      // Add label if available
      const labelTypeId = labelTypeIDs[ticket.type];
      if (labelTypeId) {
        threadInput.labelTypeIds = [labelTypeId];
      }

      // Add priority if severity is specified (Plain supports 0-3)
      if (ticket.severity) {
        const severity = parseInt(ticket.severity, 10);
        if (severity >= 0 && severity <= 3) {
          threadInput.priority = severity;
        }
      }
      if (ticket.attachmentIds && ticket.attachmentIds.length > 0) {
        threadInput.attachmentIds = ticket.attachmentIds;
      }

      let threadRes = await plainClient.createThread(threadInput);

      // Some environments can have stale/invalid label IDs. Retry once without
      // labels so ticket creation still succeeds.
      if (
        threadRes.error &&
        threadInput.labelTypeIds &&
        hasMissingLabelTypeError(threadRes.error)
      ) {
        delete threadInput.labelTypeIds;
        threadRes = await plainClient.createThread(threadInput);
      }

      if (threadRes.error) {
        console.error(
          "Error creating ticket via Plain API:",
          JSON.stringify(threadRes.error),
        );
        return {
          success: false,
          error: threadRes.error.message,
        };
      }

      return {
        success: true,
        threadId: threadRes.data.id,
      };
    } catch (error) {
      console.error("Error creating ticket:", error);
      return {
        success: false,
        error:
          error instanceof Error ? error.message : "Failed to create ticket",
      };
    }
  });

// Customer tier types
export type CustomerTierInfo = {
  customerId?: string;
  companyId?: string;
  companyName?: string;
  tierId?: string;
  tierName?: string;
  tierExternalId?: string;
  isEnterprise: boolean;
  isPaid: boolean;
  hasPremiumSupport: boolean;
};

type CustomerWithCompanyResponse = {
  customerByEmail: {
    id: string;
    company: {
      id: string;
      name: string;
      tier: {
        id: string;
        name: string;
        externalId: string | null;
      } | null;
    } | null;
  } | null;
};

export const getCustomerTierByEmail = createServerFn({ method: "GET" })
  .middleware([authMiddleware])
  .handler(async ({ context }): Promise<CustomerTierInfo> => {
    try {
      const email = context.userEmail;

      const res = (await plainClient.rawRequest({
        query: `
          query GetCustomerTier($email: String!) {
            customerByEmail(email: $email) {
              id
              company {
                id
                name
                tier {
                  id
                  name
                  externalId
                }
              }
            }
          }
        `,
        variables: {
          email,
        },
      })) as { data: CustomerWithCompanyResponse; error?: PlainSDKError };

      if (res.error) {
        return {
          isEnterprise: false,
          isPaid: false,
          hasPremiumSupport: false,
        };
      }

      const customer = res.data.customerByEmail;
      if (!customer) {
        return {
          isEnterprise: false,
          isPaid: false,
          hasPremiumSupport: false,
        };
      }
      const company = customer.company;
      const tier = company?.tier;

      // Check tier name/externalId for Enterprise status
      // Adjust these checks based on your actual tier naming conventions
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
      const tierName = tier?.name?.toLowerCase() || "";
      const tierExternalId = tier?.externalId?.toLowerCase() || "";
      const isEnterprise =
        tierExternalId.includes("enterprise") ||
        tierName.toLowerCase().includes("enterprise");
      const hasPremiumSupport = tierExternalId.includes("premium_support");
      const isPaid =
        isEnterprise ||
        hasPremiumSupport ||
        tierExternalId.includes("pro") ||
        tierExternalId.includes("vip");

      return {
        customerId: customer.id,
        companyId: company?.id,
        companyName: company?.name,
        tierId: tier?.id,
        tierName: tier?.name,
        tierExternalId: tier?.externalId || undefined,
        isEnterprise,
        isPaid,
        hasPremiumSupport,
      };
    } catch (error) {
      console.error("Error fetching customer tier:", error);
      return {
        isEnterprise: false,
        isPaid: false,
        hasPremiumSupport: false,
      };
    }
  });

// Close ticket (mark thread as done)
export type CloseTicketResult = {
  success: boolean;
  error?: string;
};

export const closeTicket = createServerFn({ method: "POST" })
  .middleware([authMiddleware])
  .inputValidator((data: { threadId: string }) => data)
  .handler(async ({ data, context }): Promise<CloseTicketResult> => {
    try {
      const authEmail = context.userEmail;
      const { threadId } = data;

      // Get the thread to verify ownership and find customer ID for the note
      const threadRes = (await plainClient.rawRequest({
        query: `
          query GetThreadCustomer($threadId: ID!) {
            thread(threadId: $threadId) {
              customer {
                id
                email {
                  email
                }
              }
            }
          }
        `,
        variables: { threadId },
      })) as {
        data?: {
          thread: { customer: { id: string; email: { email: string } } };
        };
        error?: PlainSDKError;
      };

      if (threadRes.error || !threadRes.data) {
        console.error("Error fetching thread for note:", threadRes.error);
        return {
          success: false,
          error: "Failed to fetch ticket details",
        };
      }

      // Verify the authenticated user owns this thread
      if (
        threadRes.data.thread.customer.email.email.toLowerCase() !==
        authEmail.toLowerCase()
      ) {
        return {
          success: false,
          error: "Not authorized to close this ticket",
        };
      }

      const customerId = threadRes.data.thread.customer.id;

      // Add a note to the thread before closing
      const noteRes = (await plainClient.rawRequest({
        query: `
          mutation CreateNote($input: CreateNoteInput!) {
            createNote(input: $input) {
              note {
                id
              }
              error {
                message
                type
                code
              }
            }
          }
        `,
        variables: {
          input: {
            threadId,
            customerId,
            text: "User closed the ticket",
            markdown: "User closed the ticket",
          },
        },
      })) as {
        data?: {
          createNote: {
            note: { id: string } | null;
            error: { message: string } | null;
          };
        };
        error?: PlainSDKError;
      };

      if (noteRes.error) {
        console.error("Error adding close note:", noteRes.error);
        // Continue to close even if note fails — closing is the primary action
      } else if (noteRes.data?.createNote.error) {
        console.error(
          "Error adding close note:",
          noteRes.data.createNote.error,
        );
      }

      // Mark the thread as done
      const res = await plainClient.markThreadAsDone({ threadId });

      if (res.error) {
        console.error("Error closing ticket:", res.error);
        return {
          success: false,
          error: res.error.message,
        };
      }

      return {
        success: true,
      };
    } catch (error) {
      console.error("Error closing ticket:", error);
      return {
        success: false,
        error:
          error instanceof Error ? error.message : "Failed to close ticket",
      };
    }
  });

// --- Attachment upload ---

// Allowed MIME types for attachment uploads
const ALLOWED_MIME_TYPES = new Set([
  // Images
  "image/jpeg",
  "image/png",
  "image/gif",
  "image/webp",
  "image/svg+xml",
  "image/bmp",
  "image/tiff",
  // Documents
  "application/pdf",
  "text/plain",
  "text/csv",
  "text/markdown",
  "application/rtf",
  // Microsoft Office
  "application/msword",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "application/vnd.ms-excel",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
  "application/vnd.ms-powerpoint",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  // Archives
  "application/zip",
  "application/x-zip-compressed",
  "application/gzip",
  // Video
  "video/quicktime",
]);

// Blocked file extensions (double-check even if MIME looks safe)
const BLOCKED_EXTENSIONS = new Set([
  ".exe",
  ".bat",
  ".cmd",
  ".com",
  ".msi",
  ".scr",
  ".pif",
  ".ps1",
  ".vbs",
  ".vbe",
  ".js",
  ".jse",
  ".ws",
  ".wsf",
  ".wsc",
  ".wsh",
  ".sh",
  ".bash",
  ".csh",
  ".ksh",
  ".dll",
  ".sys",
  ".drv",
  ".app",
  ".dmg",
  ".pkg",
  ".deb",
  ".rpm",
  ".iso",
  ".jar",
  ".class",
  ".php",
  ".py",
  ".rb",
  ".pl",
  ".cgi",
  ".asp",
  ".aspx",
  ".reg",
  ".inf",
  ".lnk",
  ".hta",
  ".htm",
  ".html",
  ".svg", // allow svg via MIME but block raw .svg extension spoofing
]);

export const MAX_ATTACHMENT_SIZE_BYTES = 10 * 1024 * 1024; // 10 MB
export const MAX_ATTACHMENTS = 5;

/**
 * Validates a file name and MIME type against the allow/block lists.
 * Returns an error message if invalid, or null if valid.
 */
export function validateAttachment(
  fileName: string,
  mimeType: string,
  fileSizeBytes: number,
): string | null {
  // Check file size
  if (fileSizeBytes > MAX_ATTACHMENT_SIZE_BYTES) {
    return `File "${fileName}" exceeds the maximum size of 10 MB.`;
  }

  // Allow empty MIME (some browsers don't report it) and rely on extension check
  if (mimeType && !ALLOWED_MIME_TYPES.has(mimeType)) {
    return `File type "${mimeType}" is not allowed. Please upload a common document or image file.`;
  }

  // Check extension against blocklist
  const ext =
    fileName.lastIndexOf(".") !== -1
      ? fileName.slice(fileName.lastIndexOf(".")).toLowerCase()
      : "";
  if (ext && BLOCKED_EXTENSIONS.has(ext)) {
    return `File extension "${ext}" is not allowed for security reasons.`;
  }

  return null;
}

// Accept string for the file input element
export const ACCEPTED_FILE_TYPES = [
  ".pdf",
  ".txt",
  ".csv",
  ".md",
  ".rtf",
  ".doc",
  ".docx",
  ".xls",
  ".xlsx",
  ".ppt",
  ".pptx",
  ".zip",
  ".gz",
  ".mov",
  ".jpg",
  ".jpeg",
  ".png",
  ".gif",
  ".webp",
  ".bmp",
  ".tiff",
].join(",");

export type AttachmentUploadUrlResult = {
  success: boolean;
  uploadFormUrl?: string;
  uploadFormData?: Array<{ key: string; value: string }>;
  attachmentId?: string;
  error?: string;
};

export type AttachmentUploadContext = "chat" | "customEntry";

export const getUploadUrl = createServerFn({ method: "POST" })
  .middleware([authMiddleware])
  .inputValidator(
    (data: {
      fileName: string;
      fileSizeBytes: number;
      context?: AttachmentUploadContext;
    }) => data,
  )
  .handler(async ({ data, context }): Promise<AttachmentUploadUrlResult> => {
    try {
      const userEmail = context.userEmail;
      const { fileName, fileSizeBytes } = data;
      const uploadContext = data.context || "chat";

      // Server-side validation
      // We use a generic MIME check based on extension since we don't have the real MIME here
      const ext =
        fileName.lastIndexOf(".") !== -1
          ? fileName.slice(fileName.lastIndexOf(".")).toLowerCase()
          : "";
      if (ext && BLOCKED_EXTENSIONS.has(ext)) {
        return {
          success: false,
          error: `File extension "${ext}" is not allowed for security reasons.`,
        };
      }
      if (fileSizeBytes > MAX_ATTACHMENT_SIZE_BYTES) {
        return {
          success: false,
          error: `File exceeds the maximum size of 10 MB.`,
        };
      }

      // Get customer ID
      const customer = await plainClient.getCustomerByEmail({
        email: userEmail,
      });
      if (customer.error || !customer.data) {
        return { success: false, error: "Could not find customer account." };
      }

      const attachmentType =
        uploadContext === "customEntry"
          ? AttachmentType.CustomTimelineEntry
          : AttachmentType.Chat;

      const res = await plainClient.createAttachmentUploadUrl({
        customerId: customer.data.id,
        fileName,
        fileSizeBytes,
        attachmentType,
      });

      if (res.error) {
        console.error("Error creating attachment upload URL:", res.error);
        return { success: false, error: res.error.message };
      }

      return {
        success: true,
        uploadFormUrl: res.data.uploadFormUrl,
        uploadFormData: res.data.uploadFormData,
        attachmentId: res.data.attachment.id,
      };
    } catch (error) {
      console.error("Error getting attachment upload URL:", error);
      return {
        success: false,
        error:
          error instanceof Error ? error.message : "Failed to get upload URL",
      };
    }
  });

// Reply to thread types
export type ReplyToThreadInput = {
  threadId: string;
  message: string;
  /** @deprecated No longer used - email is derived from authenticated session */
  userEmail?: string;
  /** Optional attachment IDs to include with the reply */
  attachmentIds?: string[];
};

export type ReplyToThreadResult = {
  success: boolean;
  error?: string;
};

export const replyToThread = createServerFn({ method: "POST" })
  .middleware([authMiddleware])
  .inputValidator((data: ReplyToThreadInput) => data)
  .handler(async ({ data, context }): Promise<ReplyToThreadResult> => {
    try {
      const { threadId, message, attachmentIds } = data;
      const userEmail = context.userEmail;

      // Verify the user owns this thread before allowing a reply
      const threadRes = (await plainClient.rawRequest({
        query: `
          query GetThreadCustomer($threadId: ID!) {
            thread(threadId: $threadId) {
              customer {
                email {
                  email
                }
              }
            }
          }
        `,
        variables: { threadId },
      })) as unknown as Result<
        { thread: { customer: { email: { email: string } } } },
        PlainSDKError
      >;

      if (threadRes.error) {
        return { success: false, error: "Failed to verify thread ownership" };
      }

      if (
        threadRes.data.thread.customer.email.email.toLowerCase() !==
        userEmail.toLowerCase()
      ) {
        return {
          success: false,
          error: "Not authorized to reply to this thread",
        };
      }

      const input: {
        threadId: string;
        textContent: string;
        markdownContent: string;
        attachmentIds?: string[];
        impersonation: {
          asCustomer: {
            customerIdentifier: {
              emailAddress: string;
            };
          };
        };
      } = {
        threadId,
        textContent: message,
        markdownContent: message,
        impersonation: {
          asCustomer: {
            customerIdentifier: {
              emailAddress: userEmail,
            },
          },
        },
      };
      if (attachmentIds && attachmentIds.length > 0) {
        input.attachmentIds = attachmentIds;
      }

      const res = (await plainClient.rawRequest({
        query: `
          mutation ReplyToThread($input: ReplyToThreadInput!) {
            replyToThread(input: $input) {
              error {
                message
                type
                code
              }
            }
          }
        `,
        variables: { input },
      })) as {
        data: {
          replyToThread: {
            error: { message: string; type: string; code: string } | null;
          };
        };
        error?: PlainSDKError;
      };

      if (res.error) {
        console.error("Error replying to thread:", res.error);
        return {
          success: false,
          error: res.error.message,
        };
      }

      if (res.data.replyToThread.error) {
        console.error(
          "Error replying to thread:",
          res.data.replyToThread.error,
        );
        return {
          success: false,
          error: res.data.replyToThread.error.message,
        };
      }

      return {
        success: true,
      };
    } catch (error) {
      console.error("Error replying to thread:", error);
      return {
        success: false,
        error:
          error instanceof Error ? error.message : "Failed to send message",
      };
    }
  });
