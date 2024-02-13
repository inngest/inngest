export function formatXAxis(value: unknown): string {
  // Should be impossible, but "value" isn't typed so it's good to check.
  if (!(value instanceof Date)) {
    return '';
  }

  const dayOfMonth = value.getUTCDate();

  // Only show every 7 days starting with 1 (1, 8, 15, etc.).
  if ((dayOfMonth - 1) % 7 !== 0) {
    return '';
  }

  let suffix = '';
  if (dayOfMonth === 1) {
    suffix = 'st';
  } else if ([8, 15, 29].includes(dayOfMonth)) {
    suffix = 'th';
  } else if (dayOfMonth === 22) {
    suffix = 'nd';
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
