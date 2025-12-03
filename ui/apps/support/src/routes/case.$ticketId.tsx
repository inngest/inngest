import { createFileRoute, Link } from "@tanstack/react-router";
import { getTicketById, type TicketDetail } from "@/data/plain";
import { RiArrowLeftLine } from "@remixicon/react";

export const Route = createFileRoute("/case/$ticketId")({
  component: TicketDetail,
  loader: async ({ params }) => {
    const ticket = await getTicketById({ data: { ticketId: params.ticketId } });

    if (!ticket) {
      throw new Error("Ticket not found");
    }

    return {
      ticket,
    };
  },
});

function TicketDetail() {
  const { ticket } = Route.useLoaderData();

  const getStatusColor = (status: string) => {
    const statusStr = status ? String(status).toLowerCase() : "";
    switch (statusStr) {
      case "todo":
        return "bg-yellow-100 text-yellow-800";
      case "done":
        return "bg-green-100 text-green-800";
      case "snoozed":
        return "bg-blue-100 text-blue-800";
      default:
        return "bg-gray-100 text-gray-800";
    }
  };

  const getPriorityColor = (priority: string) => {
    const priorityStr = priority ? String(priority).toLowerCase() : "";
    switch (priorityStr) {
      case "urgent":
        return "text-red-600 bg-red-50";
      case "high":
        return "text-orange-600 bg-orange-50";
      case "normal":
        return "text-blue-600 bg-blue-50";
      case "low":
        return "text-gray-600 bg-gray-50";
      default:
        return "text-gray-600 bg-gray-50";
    }
  };

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleString(undefined, {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "numeric",
      minute: "2-digit",
    });
  };

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
          <span
            className={`rounded-full px-3 py-1 text-sm font-medium ${getStatusColor(
              ticket.status,
            )}`}
          >
            {ticket.status}
          </span>
          <span
            className={`rounded-full px-3 py-1 text-sm font-medium ${getPriorityColor(
              ticket.priority,
            )}`}
          >
            {ticket.priority} priority
          </span>
        </div>

        <div className="text-muted mt-4 space-y-1 text-sm">
          <div>
            <span className="font-medium">Customer:</span> {ticket.customerName}
          </div>
          <div>
            <span className="font-medium">Created:</span>{" "}
            {formatTimestamp(ticket.createdAt)}
          </div>
          <div>
            <span className="font-medium">Last updated:</span>{" "}
            {formatTimestamp(ticket.updatedAt)}
          </div>
        </div>

        {ticket.description && (
          <div className="bg-canvasSubtle mt-4 rounded-lg p-4">
            <p className="text-basis text-sm">{ticket.description}</p>
          </div>
        )}
      </div>

      {/* Conversation timeline */}
      <div className="space-y-4">
        <h2 className="text-basis text-lg font-semibold">Conversation</h2>

        {ticket.timelineEntries.length === 0 ? (
          <div className="bg-canvasSubtle text-muted rounded-lg border border-subtle p-8 text-center">
            <p>No messages in this conversation yet.</p>
          </div>
        ) : (
          <div className="space-y-4">
            {ticket.timelineEntries.map((entry) => (
              <div
                key={entry.id}
                className={`rounded-lg border p-4 ${
                  entry.actorType === "customer"
                    ? "bg-blue-50 border-blue-200"
                    : "bg-canvasBase border-subtle"
                }`}
              >
                <div className="mb-2 flex items-start justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-basis font-medium">
                      {entry.actorName}
                    </span>
                    <span
                      className={`rounded px-2 py-0.5 text-xs ${
                        entry.actorType === "customer"
                          ? "bg-blue-100 text-blue-700"
                          : entry.actorType === "user"
                          ? "bg-green-100 text-green-700"
                          : "bg-gray-100 text-gray-700"
                      }`}
                    >
                      {entry.actorType}
                    </span>
                  </div>
                  <span className="text-muted text-xs">
                    {formatTimestamp(entry.timestamp)}
                  </span>
                </div>

                {entry.title && (
                  <h3 className="text-basis mb-2 font-medium">{entry.title}</h3>
                )}

                {entry.text && (
                  <div className="text-basis whitespace-pre-wrap text-sm">
                    {entry.text}
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
