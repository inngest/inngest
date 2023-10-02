// Format: 20 Jul 2023, 00:08:42
export function fullDate(date: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(date);
}

// Format: 20/07/2023, 00:08:42
export function shortDate(date: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'short',
    timeStyle: 'medium',
  }).format(date);
}

export function formatMilliseconds(durationInMs): string {
  if (durationInMs >= 3600000) { // 1 hour = 3600000 ms
    const hours = Math.floor(durationInMs / 3600000);
    durationInMs %= 3600000;
    
    const minutes = Math.floor(durationInMs / 60000);
    return `${hours}h ${minutes}m`;
  } else if (durationInMs >= 60000) { // 1 minute = 60000 ms
    const minutes = Math.floor(durationInMs / 60000);
    durationInMs %= 60000;

    const seconds = Math.floor(durationInMs / 1000);
    return `${minutes}m ${seconds}s`;
  } else if (durationInMs >= 1000) { // 1 second = 1000 ms
    const seconds = Math.floor(durationInMs / 1000);
    return `${seconds}s`;
  } else {
    return `${durationInMs}ms`;
  }
}
