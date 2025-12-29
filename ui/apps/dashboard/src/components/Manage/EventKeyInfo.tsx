import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link/NewLink';

export const EventKeyInfo = () => (
  <Info
    text="Event keys are unique keys that allow applications to send Inngest events."
    action={
      <Link
        href={'https://www.inngest.com/docs/events/creating-an-event-key'}
        target="_blank"
      >
        Learn more
      </Link>
    }
  />
);
