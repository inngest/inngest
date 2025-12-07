import * as Sentry from "@sentry/tanstackstart-react";

Sentry.init({
  dsn: import.meta.env.VITE_SENTRY_DSN,
  tracesSampleRate: 0.2,
});
