import { useState } from "react";
import {
  RiCheckboxCircleFill,
  RiArrowDownSLine,
  RiAlertFill,
  RiExternalLinkLine,
} from "@remixicon/react";
import type {
  ExtendedStatus,
  Event,
  MaintenanceScheduledEvent,
} from "./Status";
import {
  impactSchema,
  type Impact,
} from "@inngest/components/SharedContext/useInngestStatus";
import { formatDistanceToNow } from "date-fns";
import { Link } from "@inngest/components/Link";

type StatusBannerProps = {
  status?: ExtendedStatus;
};

export function StatusBanner({ status }: StatusBannerProps) {
  if (!status) return null;

  const isOperational = status.impact === "none";
  const isOutage = impactSchema.options.includes(status.impact as Impact);
  const isMaintenance = status.impact === "maintenance";
  const isMaintenanceScheduled = status.scheduled_maintenances.length > 0;

  const [isOpen, setIsOpen] = useState(!isOperational);

  return (
    <div
      className={`flex flex-col gap-2 w-full items-start px-4 py-2 ${
        isOutage
          ? "bg-error text-error"
          : isMaintenance
          ? "bg-info text-info"
          : "bg-success text-success"
      }`}
    >
      <div className="flex flex-1 w-full items-center justify-between">
        <div className="flex items-center gap-2">
          {isOutage || isMaintenance ? (
            <RiAlertFill className="h-4 w-4" />
          ) : (
            <RiCheckboxCircleFill className="h-4 w-4" />
          )}
          <p className="text-sm font-medium">{status.description}</p>
          {isMaintenanceScheduled && (
            <p className="text-sm">Upcoming maintenance scheduled</p>
          )}
        </div>
        <button
          onClick={() => setIsOpen(!isOpen)}
          className="p-2 cursor-pointer"
          aria-label="Toggle status banner"
          title="Toggle status banner"
        >
          <RiArrowDownSLine
            className={`h-4 w-4 text-muted ${isOpen ? "rotate-180" : ""}`}
          />
        </button>
      </div>
      <div
        className={`flex flex-col items-start pt-2 pb-1 gap-3 ${
          isOpen ? "block" : "hidden"
        }`}
      >
        {status.incidents.length > 0 &&
          status.incidents.map((incident) => (
            <Event key={incident.id} event={incident} />
          ))}
        {status.maintenances.length > 0 &&
          status.maintenances.map((maintenance) => (
            <Event key={maintenance.id} event={maintenance} />
          ))}
        {status.scheduled_maintenances.length > 0 &&
          status.scheduled_maintenances.map((maintenance) => (
            <Event key={maintenance.id} event={maintenance} />
          ))}
        <Link
          href="https://status.inngest.com"
          target="_blank"
          rel="noopener noreferrer"
          iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
        >
          View status page
        </Link>
      </div>
    </div>
  );
}

function Event({ event }: { event: Event }) {
  return (
    <div key={event.id} className="flex flex-col gap-2 text-sm">
      <p className="font-medium">
        {event.name}
        {event.status === "maintenance_scheduled" &&
          (event as MaintenanceScheduledEvent)?.starts_at && (
            <>
              {` - ${formatDistanceToNow(new Date(event.starts_at), {
                addSuffix: true,
              })} `}
              <span className="font-mono">({event.starts_at})</span>
            </>
          )}
      </p>
      <p className="text-muted">{event.last_update_message}</p>
    </div>
  );
}
