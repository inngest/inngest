import { PlainClient } from "@team-plain/typescript-sdk/dist/index";
import { createServerFn } from "@tanstack/react-start";

// Initialize Plain client
// The API key should be set in the environment variable PLAIN_API_KEY
const plainClient = new PlainClient({
  apiKey: process.env.PLAIN_API_KEY || "",
});

export type TicketSummary = {
  id: string;
  title: string;
  status: string;
  priority: string;
  createdAt: string;
  updatedAt: string;
};

export type TimelineEntry = {
  id: string;
  timestamp: string;
  actorName: string;
  actorType: string;
  text?: string;
  title?: string;
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
  timelineEntries: TimelineEntry[];
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

      // For now, return basic thread details without timeline
      // Timeline fetching can be added once we have the proper SDK methods
      const timelineEntries: TimelineEntry[] = [];

      return {
        id: thread.id,
        title: thread.title || "Untitled",
        description: thread.description || null,
        status: String(thread.status || "UNKNOWN"),
        priority: String(thread.priority || "NORMAL"),
        createdAt: thread.createdAt?.iso8601 || new Date().toISOString(),
        updatedAt: thread.updatedAt?.iso8601 || new Date().toISOString(),
        customerName: thread.customer?.email || "Unknown",
        timelineEntries,
      };
    } catch (error) {
      console.error("Error fetching ticket details:", error);
      return null;
    }
  });
