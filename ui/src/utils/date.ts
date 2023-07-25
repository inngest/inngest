// Format: 20 Jul 2023, 00:08:42
export function fullDate(date: Date): string {
  return new Intl.DateTimeFormat('en-UK', {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(date);
}
