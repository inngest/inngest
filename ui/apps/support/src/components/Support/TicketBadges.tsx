import { getLabelForStatus } from "@/data/plain";
import { getStatusColor, getPriorityColor } from "@/utils/ticket";

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
 */
export function StatusBadge({ status, size = "md" }: StatusBadgeProps) {
  const sizeClasses = size === "sm" ? "px-2 py-1 text-xs" : "px-3 py-1 text-sm";

  return (
    <span
      className={`rounded-full font-medium ${sizeClasses} ${getStatusColor(
        status,
      )}`}
    >
      {getLabelForStatus(status)}
    </span>
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
