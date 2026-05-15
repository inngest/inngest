import SegmentedControl from "@inngest/components/SegmentedControl/SegmentedControl";
import {
  TICKET_STATUS_ALL,
  TICKET_STATUS_OPEN,
  TICKET_STATUS_CLOSED,
  type TicketStatusFilter,
} from "@/data/plain";

type FiltersProps = {
  status: TicketStatusFilter | undefined;
  onStatusChange: (status: TicketStatusFilter | undefined) => void;
  defaultStatus: TicketStatusFilter | undefined;
};

export function Filters({
  status,
  onStatusChange,
  defaultStatus = TICKET_STATUS_OPEN,
}: FiltersProps) {
  return (
    <div className="flex flex-row">
      <SegmentedControl defaultValue={status ?? defaultStatus}>
        <SegmentedControl.Button
          value={TICKET_STATUS_ALL}
          onClick={() => onStatusChange(TICKET_STATUS_ALL)}
        >
          All
        </SegmentedControl.Button>
        <SegmentedControl.Button
          value={TICKET_STATUS_OPEN}
          onClick={() => onStatusChange(TICKET_STATUS_OPEN)}
        >
          Open
        </SegmentedControl.Button>
        <SegmentedControl.Button
          value={TICKET_STATUS_CLOSED}
          onClick={() => onStatusChange(TICKET_STATUS_CLOSED)}
        >
          Closed
        </SegmentedControl.Button>
      </SegmentedControl>
    </div>
  );
}
