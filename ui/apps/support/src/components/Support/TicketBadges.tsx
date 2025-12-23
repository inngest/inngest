import { getLabelForStatus } from "@/data/plain";
import { getPriorityColor } from "@/utils/ticket";
import { Pill } from "@inngest/components/Pill";

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
 * Uses Pill component with "primary" for open status and "info" for closed status
 */
export function StatusBadge({ status, size = "md" }: StatusBadgeProps) {
  const statusStr = status ? String(status).toLowerCase() : "";
  const label = getLabelForStatus(status);

  // Map status to Pill kind: "primary" for open, "info" for closed
  let pillKind: "primary" | "info" = "primary";
  if (statusStr === "done") {
    pillKind = "info";
  }

  return (
    <Pill kind={pillKind} appearance="solid">
      {label}
    </Pill>
  );
}

/**
 * Badge component for displaying ticket priority
 */
export function PriorityBadge({
  priority,
  size = "md",
  showLabel = true,
}: PriorityBadgeProps) {
  const sizeClasses = size === "sm" ? "px-2 py-1 text-xs" : "px-3 py-1 text-sm";

  return (
    <span
      className={`rounded-full font-medium ${sizeClasses} ${getPriorityColor(
        priority,
      )}`}
    >
      P{priority}
      {showLabel && " priority"}
    </span>
  );
}
