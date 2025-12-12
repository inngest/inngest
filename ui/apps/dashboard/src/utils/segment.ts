import { AnalyticsBrowser } from '@segment/analytics-next';

let analyticsInstance: ReturnType<typeof AnalyticsBrowser.load> | undefined;

export const analytics = new Proxy(
  {} as ReturnType<typeof AnalyticsBrowser.load>,
  {
    get: (_target, prop) => {
      if (typeof window === 'undefined') {
        return () => {};
      }

      if (!analyticsInstance) {
        analyticsInstance = AnalyticsBrowser.load(
          {
            writeKey: import.meta.env.VITE_SEGMENT_WRITE_KEY!,
            cdnURL:
              process.env.NODE_ENV === 'production'
                ? 'https://analytics-cdn.inngest.com'
                : undefined,
          },
          {
            integrations: {
              'Segment.io': {
                apiHost:
                  process.env.NODE_ENV === 'production'
                    ? 'analytics.inngest.com/v1'
                    : undefined,
                protocol: 'https',
              },
            },
          },
        );
      }

      return analyticsInstance[prop as keyof typeof analyticsInstance];
    },
  },
);
