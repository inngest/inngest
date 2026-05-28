import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';

export const SessionsInfo = () => (
  <Info
    text="Sessions group runs that share a session ID, sent on an event's sessions field as a map of session keys to session IDs."
    action={
      <Link href="https://www.inngest.com/docs" target="_blank">
        Learn about sessions
      </Link>
    }
  />
);
