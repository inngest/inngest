import { createFileRoute, Link } from "@tanstack/react-router";
import {
  getTicketById,
  getTimelineEntriesForTicket,
  type TicketDetail,
} from "@/data/plain";
import { RiArrowLeftLine } from "@remixicon/react";
import { Markdown } from "@/components/Markdown/Markdown";
import { StatusBadge, PriorityBadge } from "@/components/Support/TicketBadges";
import { formatTimestamp } from "@/utils/ticket";

export const Route = createFileRoute("/_authed/case/$ticketId")({
  component: TicketDetail,
  loader: async ({ params }) => {
    const [ticket, timelineEntries] = await Promise.all([
      getTicketById({ data: { ticketId: params.ticketId as string } }),
      getTimelineEntriesForTicket({
        data: { ticketId: params.ticketId as string },
      }),
    ]);

    return { ticket, timelineEntries };
  },
});

function TicketDetail() {
  const { ticket, timelineEntries } = Route.useLoaderData();

  if (!ticket || !timelineEntries) {
    return <div>Error loading ticket</div>;
  }

  return (
    <div className="px-6 pt-4 pb-8">
      {/* Back button */}
      <Link
        to="/"
        className="text-muted hover:text-basis mb-4 inline-flex items-center gap-1 text-sm"
      >
        <RiArrowLeftLine className="h-4 w-4" />
        Back to tickets
      </Link>

      {/* Ticket header */}
      <div className="border-subtle mb-6 border-b pb-6">
        <h1 className="text-basis mb-3 text-2xl font-semibold">
          {ticket.title}
        </h1>

        <div className="flex items-center gap-3">
          <StatusBadge status={ticket.status} />
          <PriorityBadge priority={ticket.priority} showLabel={false} />
        </div>

        <div className="text-muted mt-4 space-y-1 text-sm">
          <div>
            <span className="font-medium">Created:</span>{" "}
            {formatTimestamp(ticket.createdAt)}
          </div>
          <div>
            <span className="font-medium">Last updated:</span>{" "}
            {formatTimestamp(ticket.updatedAt)}
          </div>
        </div>
      </div>

      {/* Conversation timeline */}
      <div className="space-y-4">
        <h2 className="text-basis text-lg font-semibold">Conversation</h2>

        {timelineEntries.length === 0 ? (
          <div className="bg-canvasSubtle text-muted rounded-lg border border-subtle p-8 text-center">
            <p>No messages in this conversation yet.</p>
          </div>
        ) : (
          <div className="space-y-4">
            {timelineEntries.map((entry) => (
              <div
                key={entry.node.id}
                className={`rounded-lg border p-4 ${
                  entry.node.actor.__typename === "CustomerActor"
                    ? "bg-blue-50 border-blue-200"
                    : "bg-canvasBase border-subtle"
                }`}
              >
                <div className="mb-2 flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-basis font-medium">
                      {entry.node.actor.__typename === "CustomerActor"
                        ? entry.node.actor.customer.fullName
                        : entry.node.actor.__typename === "UserActor"
                        ? "Inngest Support"
                        : entry.node.actor.__typename === "MachineUserActor"
                        ? "System"
                        : "Unknown"}
                    </span>
                  </div>
                  <span className="text-muted text-xs">
                    {formatTimestamp(entry.node.timestamp.iso8601)}
                  </span>
                </div>
                {entry.node.entry.__typename === "EmailEntry" && (
                  <div className="mt-4">
                    <Markdown content={entry.node.entry.markdownContent} />
                  </div>
                )}
                {entry.node.entry.__typename === "CustomEntry" && (
                  <div className="mt-4">
                    <Markdown
                      content={entry.node.entry.components
                        .map((component) => component.text)
                        .join("\n")}
                    />
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
