export function formatXAxis(value: unknown): string {
  // Should be impossible, but "value" isn't typed so it's good to check.
  if (!(value instanceof Date)) {
    return '';
  }

  const dayOfMonth = value.getUTCDate();

  // Only show every 3 days starting with 1 (1, 3, 5, etc.).
  if (dayOfMonth % 2 === 0) {
    return '';
  }

  let suffix = 'th';
  if (dayOfMonth % 10 === 1 && dayOfMonth !== 11) {
    suffix = 'st';
  } else if (dayOfMonth % 10 === 2 && dayOfMonth !== 12) {
    suffix = 'nd';
  } else if (dayOfMonth % 10 === 3 && dayOfMonth !== 13) {
    suffix = 'rd';
  }

  return `${dayOfMonth}${suffix}`;
}

export function formatYAxis(value: unknown): string {
  // Should be impossible, but "value" isn't typed so it's good to check.
  if (typeof value !== 'number' || value === 0) {
    return '';
  }

  if (value >= 1000) {
    return `${value / 1000}k`;
  }

  return value.toString();
}

// Formats a date function to a string in the format "January 2, 2023". Will use
// the UTC timezone.
export function toLocaleUTCDateString(date: Date): string {
  return date.toLocaleDateString(undefined, {
    day: 'numeric',
    month: 'long',
    timeZone: 'UTC',
    year: 'numeric',
  });
}
