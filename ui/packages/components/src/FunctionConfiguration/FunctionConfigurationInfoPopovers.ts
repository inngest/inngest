export const PopoverContent = {
  failure: {
    text: 'Another function invoked when this function has exhausted all of its retries.',
    url: 'https://www.inngest.com/docs/features/inngest-functions/error-retries/failure-handlers',
  },
  cancelOn: {
    text: 'Prevent unnecessary actions based on previous actions or stop an unwanted function run composed of multiple steps.',
    url: 'https://www.inngest.com/docs/features/inngest-functions/cancellation',
  },
  retries: {
    text: 'The number of times the function will be retried when it errors.',
    url: 'https://www.inngest.com/docs/features/inngest-functions/error-retries/retries',
  },
  rateLimit: {
    text: 'A hard limit on how many function runs can start within a time period. Events that exceed the rate limit are skipped and do not trigger functions to start.',
    url: 'https://www.inngest.com/docs/guides/rate-limiting',
  },
  debounce: {
    text: 'Delays function execution until a series of events are no longer received.',
    url: 'https://www.inngest.com/docs/guides/debounce',
  },
  priority: {
    text: 'When the function is triggered multiple times simultaneously, the priority determines the order in which they are run.\n\n The priority value is determined by evaluating the configured expression. The higher the value, the higher the priority.',
    url: 'https://www.inngest.com/docs/guides/priority',
  },
  batching: {
    text: 'Process multiple events in a single function run.',
    url: 'https://www.inngest.com/docs/guides/batching',
  },
  singleton: {
    text: 'Ensure that only a single run of your function (or a set of specific function runs, based on specific event properties) is happening at a time.',
    url: 'https://www.inngest.com/docs/guides/singleton',
  },
  concurrency: {
    text: 'The maximum number of concurrently running steps.',
    url: 'https://www.inngest.com/docs/guides/concurrency',
  },
  throttle: {
    text: 'How many function runs can start within a time period. When the limit is reached, new function runs over the throttling limit will be enqueued for the future.',
    url: 'https://www.inngest.com/docs/guides/throttling',
  },
};

export type InfoPopoverContent = {
  text: string;
  url: string;
};
