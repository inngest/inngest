export function entitlementSecondsToStr(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m == 0) {
    return `${s} sec`;
  }
  if (s == 0) {
    return `${m} mins`;
  }
  return `${m} mins ${s} sec`;
}
