import { Pill } from "@inngest/components/Pill";
import { type PillKind } from "@inngest/components/Pill/Pill";

type StatusBadgeProps = {
  status: string;
  size?: "sm" | "md";
};

type PriorityBadgeProps = {
  priority: string;
  size?: "sm" | "md";
  showLabel?: boolean;
};

/**
 * Badge component for displaying ticket status
 * Uses Pill component with "secondary" for open status and "primary" for completed status
 */
export function StatusBadge({ status }: StatusBadgeProps) {
  const statusStr = status ? String(status).toLowerCase() : "";

  // Map status to Pill kind and label
  let pillKind: PillKind = "info";
  let label = "Open";

  if (statusStr === "done") {
    pillKind = "primary";
    label = "Completed";
  }

  return (
    <Pill kind={pillKind} appearance="solid">
      {label}
    </Pill>
  );
}

/**
 * Badge component for displaying ticket priority
 * Maps priority levels to visual styles:
 * - p0: error (red/urgent)
 * - p1: default (gray)
 * - p2+: default (gray)
 */
export function PriorityBadge({ priority }: PriorityBadgeProps) {
  const priorityNum = parseInt(String(priority), 10);

  // Map priority to Pill kind
  let pillKind: PillKind = "default";
  if (priorityNum === 0) {
    pillKind = "error";
  }

  return (
    <Pill kind={pillKind} appearance="solid">
      p{priority}
    </Pill>
  );
}
