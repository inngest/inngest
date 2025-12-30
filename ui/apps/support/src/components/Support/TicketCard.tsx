import { Link, ClientOnly } from "@tanstack/react-router";
import type { TicketSummary } from "@/data/plain";
import { StatusBadge, PriorityBadge } from "./TicketBadges";
import { ChannelBadge } from "./ChannelBadge";
import { formatDistanceToNow } from "date-fns";

type TicketCardProps = {
  ticket: TicketSummary;
};

export function TicketCard({ ticket }: TicketCardProps) {
  const timeAgo = formatDistanceToNow(new Date(ticket.updatedAt), {
    addSuffix: false,
  });

  return (
    <Link
      to="/case/$ticketId"
      params={{ ticketId: ticket.id }}
      className="border-muted hover:border-active bg-canvasBase flex flex-col md:grid md:grid-rows-3 gap-3 rounded border py-4 px-4 transition-colors text-sm md:text-base"
    >
      {/* Priority and Status Pills */}
      <div className="flex items-center gap-1.5">
        {/* Silence warning for server rendering with hook */}
        <ClientOnly>
          <PriorityBadge priority={ticket.priority} />
          <StatusBadge status={ticket.status} />
        </ClientOnly>
      </div>

      {/* Title Row */}
      <div className="flex items-center justify-between">
        <div className="text-basis flex min-w-0 flex-1 items-center gap-1.5 leading-none">
          <div className="text-muted flex flex-col justify-center">
            <p className="leading-4 whitespace-nowrap">{ticket.ref}</p>
          </div>
          <div className="flex flex-col justify-center font-medium">
            <p className="leading-4 whitespace-nowrap">{ticket.title}</p>
          </div>
        </div>

        {/* Source and Time */}
        <div className="flex items-center md:min-w-64 justify-end">
          <div className="flex md:grid md:grid-cols-2 gap-2 items-center justify-end md:gap-4">
            <ChannelBadge channel={ticket.channel} />
            <div className="text-muted flex flex-col justify-center leading-none">
              <p className="leading-4 whitespace-nowrap">{timeAgo}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Preview Text */}
      {ticket.previewText && (
        <div className="text-basis flex min-w-full flex-col justify-center">
          {/* Add padding y to not cut off text that goes below the line (g, y) */}
          <p className="leading-4 whitespace-nowrap overflow-ellipsis overflow-x-hidden py-0.5">
            {ticket.previewText}
          </p>
        </div>
      )}
    </Link>
  );
}
