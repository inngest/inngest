/**
 * Get Tailwind classes for ticket status badges
 */
export const getStatusColor = (status: string): string => {
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

/**
 * Get Tailwind classes for ticket priority badges
 */
export const getPriorityColor = (priority: string): string => {
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

/**
 * Format a timestamp into a human-readable string
 */
export const formatTimestamp = (timestamp: string): string => {
  const date = new Date(timestamp);
  return date.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
};
