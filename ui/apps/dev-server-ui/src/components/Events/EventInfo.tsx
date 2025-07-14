import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/Link';

export const EventInfo = () => (
  <Info
    text="List of all Inngest events in the development environment."
    action={
      <Link href={'https://www.inngest.com/docs/events'} target="_blank">
        Learn how events work
      </Link>
    }
  />
);
