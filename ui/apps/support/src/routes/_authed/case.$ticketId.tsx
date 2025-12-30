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
    <div className="min-h-screen bg-canvasBase">
      <div className="mx-auto max-w-4xl px-6 py-8">
        {/* Back button */}
        <Link
          to="/"
          className="text-muted hover:text-basis mb-6 inline-flex items-center gap-2 text-sm font-medium transition-colors"
        >
          <RiArrowLeftLine className="h-4 w-4" />
          Back to tickets
        </Link>

        {/* Ticket header */}
        <div className="border-subtle mb-8 rounded-xl border bg-canvasSubtle p-6">
          <h1 className="text-basis mb-4 text-2xl font-bold">{ticket.title}</h1>

          <div className="mb-4 flex flex-wrap items-center gap-3">
            <StatusBadge status={ticket.status} />
            <PriorityBadge priority={ticket.priority} showLabel={false} />
          </div>

          <div className="border-subtle flex flex-wrap gap-6 border-t pt-4 text-sm">
            <div className="text-muted">
              <span className="font-medium text-basis">Created:</span>{" "}
              {formatTimestamp(ticket.createdAt)}
            </div>
            <div className="text-muted">
              <span className="font-medium text-basis">Last updated:</span>{" "}
              {formatTimestamp(ticket.updatedAt)}
            </div>
          </div>
        </div>

        {/* Conversation timeline */}
        <div className="space-y-6">
          <div className="flex items-center gap-3">
            <h2 className="text-basis text-lg font-semibold">Conversation</h2>
            {timelineEntries.length > 0 && (
              <span className="text-muted text-sm">
                {timelineEntries.length}{" "}
                {timelineEntries.length === 1 ? "message" : "messages"}
              </span>
            )}
          </div>

          {timelineEntries.length === 0 ? (
            <div className="bg-canvasSubtle border-subtle rounded-xl border p-12 text-center">
              <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-canvasMuted">
                <svg
                  className="text-muted h-8 w-8"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
                  />
                </svg>
              </div>
              <p className="text-basis mb-1 text-lg font-medium">
                No messages yet
              </p>
              <p className="text-muted text-sm">
                The conversation will appear here once messages are exchanged.
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {timelineEntries.map((entry) => {
                const isCustomer =
                  entry.node.actor.__typename === "CustomerActor";
                const actorName =
                  entry.node.actor.__typename === "CustomerActor"
                    ? entry.node.actor.customer.fullName
                    : entry.node.actor.__typename === "UserActor"
                    ? "Inngest Support"
                    : entry.node.actor.__typename === "MachineUserActor"
                    ? "System"
                    : "Unknown";

                return (
                  <div
                    key={entry.node.id}
                    className={`flex gap-4 ${
                      isCustomer ? "flex-row-reverse" : "flex-row"
                    }`}
                  >
                    {/* Avatar */}
                    <div
                      className={`shrink-0 ${
                        isCustomer ? "order-2" : "order-1"
                      }`}
                    >
                      <div
                        className={`flex h-10 w-10 items-center justify-center rounded-full text-sm font-semibold ${
                          isCustomer
                            ? "bg-blue-100 text-blue-700"
                            : "bg-canvasMuted text-basis"
                        }`}
                      >
                        {actorName.charAt(0).toUpperCase()}
                      </div>
                    </div>

                    {/* Message content */}
                    <div
                      className={`flex-1 ${
                        isCustomer ? "order-1" : "order-2"
                      } min-w-0`}
                    >
                      <div
                        className={`rounded-lg border p-4 ${
                          isCustomer
                            ? "bg-blue-50 border-blue-200"
                            : "bg-canvasBase border-subtle"
                        }`}
                      >
                        <div className="mb-3 flex items-center justify-between">
                          <span
                            className={`font-semibold ${
                              isCustomer ? "text-blue-900" : "text-basis"
                            }`}
                          >
                            {actorName}
                          </span>
                          <span className="text-muted text-xs">
                            {formatTimestamp(entry.node.timestamp.iso8601)}
                          </span>
                        </div>
                        {entry.node.entry.__typename === "EmailEntry" && (
                          <div className="text-basis">
                            <Markdown
                              content={entry.node.entry.markdownContent}
                            />
                          </div>
                        )}
                        {entry.node.entry.__typename === "CustomEntry" && (
                          <div className="text-basis">
                            <Markdown
                              content={entry.node.entry.components
                                .map((component) => component.text)
                                .join("\n")}
                            />
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
