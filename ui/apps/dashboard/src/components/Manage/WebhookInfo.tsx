import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/NewLink';

export const WebhookInfo = () => (
  <Info
    text="Sources for events for developers."
    action={
      <Link
        href={'https://www.inngest.com/docs/platform/webhooks'}
        target="_blank"
      >
        Learn how create a webhook
      </Link>
    }
  />
);
