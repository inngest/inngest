export function entitlementSecondsToStr(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;

  let sStr = `${s} sec`;

  let mStr = `${m} mins`;
  if (m == 1) {
    mStr = `${m} min`;
  }

  if (m == 0) {
    return sStr;
  }
  if (s == 0) {
    return mStr;
  }
  return `${mStr} ${sStr}`;
}
