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

const second = 1000;
const minute = 60 * second;
const hour = 60 * minute;

export function formatMilliseconds(durationInMs): string {
  if (durationInMs >= hour) {
    const hours = Math.floor(durationInMs / hour);
    durationInMs %= hour;
    
    const minutes = Math.floor(durationInMs / minute);
    return `${hours}h ${minutes}m`;
  } else if (durationInMs >= minute) {
    const minutes = Math.floor(durationInMs / minute);
    durationInMs %= minute;

    const seconds = Math.floor(durationInMs / second);
    return `${minutes}m ${seconds}s`;
  } else if (durationInMs >= second) {
    const seconds = Math.floor(durationInMs / second);
    return `${seconds}s`;
  } else {
    return `${durationInMs}ms`;
  }
}
