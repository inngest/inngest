import { AnalyticsBrowser } from '@segment/analytics-next';

let analyticsInstance: ReturnType<typeof AnalyticsBrowser.load> | null = null;

//
// Lazy initialization to avoid hydration mismatch - only loads when first method is called (after useEffect)
export const analytics = new Proxy(
  {} as ReturnType<typeof AnalyticsBrowser.load>,
  {
    get: (_target, prop) => {
      if (typeof window === 'undefined') {
        return () => Promise.resolve();
      }

      if (analyticsInstance) {
        return analyticsInstance[prop as keyof typeof analyticsInstance];
      }

      const writeKey = import.meta.env.VITE_SEGMENT_WRITE_KEY;

      if (!writeKey) {
        console.warn(
          'VITE_SEGMENT_WRITE_KEY is not defined - segment analytics disabled',
        );
      }

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

      return analyticsInstance[prop as keyof typeof analyticsInstance];
    },
  },
);
