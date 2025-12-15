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
        const writeKey = import.meta.env.VITE_SEGMENT_WRITE_KEY;

        //
        // Only use custom CDN in production with valid writeKey
        const useCustomCdn = import.meta.env.PROD && writeKey;

        analyticsInstance = AnalyticsBrowser.load(
          {
            writeKey: writeKey!,
            cdnURL: useCustomCdn
              ? 'https://analytics-cdn.inngest.com'
              : undefined,
          },
          {
            integrations: {
              'Segment.io': {
                apiHost: useCustomCdn ? 'analytics.inngest.com/v1' : undefined,
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
