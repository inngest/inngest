import { AnalyticsBrowser } from "@segment/analytics-next";

export const analytics = AnalyticsBrowser.load(
  {
    writeKey: process.env.NEXT_PUBLIC_SEGMENT_WRITE_KEY!,
    cdnURL:
      process.env.NODE_ENV === "production"
        ? "https://analytics-cdn.inngest.com"
        : undefined,
  },
  {
    integrations: {
      "Segment.io": {
        apiHost:
          process.env.NODE_ENV === "production"
            ? "analytics.inngest.com/v1"
            : undefined,
        protocol: "https",
      },
    },
  },
);
