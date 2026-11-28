import { Banner } from "@inngest/components/Banner";

// Starts at midnight US Eastern and ends at the end of Nov 27 US Pacific so the
// whole window is covered for every US timezone.
const START = new Date("2026-11-23T00:00:00-05:00");
const END = new Date("2026-11-28T00:00:00-08:00");

export function HolidayBanner() {
  const now = new Date();
  if (now < START || now >= END) return null;

  return (
    <Banner severity="info">
      Happy Thanksgiving! Our US team is taking time off for the holiday, so
      support replies may be a little slower than usual from Nov 23 to Nov 27.
      Thanks for your patience!
    </Banner>
  );
}
