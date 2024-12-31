import { Alert } from '@inngest/components/Alert/Alert';
import { NewLink } from '@inngest/components/Link/Link';

//
// A thin wrapper around Alert for a standard error + contact support message
export const Error = ({ message, button }: { message: string; button?: React.ReactNode }) => (
  <Alert severity="error" className="mb-4 w-full" button={button}>
    {message} If the problem persists,
    <NewLink
      href="/support"
      size="medium"
      className="text-error decoration-error hover:decoration-error ml-1 inline"
    >
      contact support
    </NewLink>
    .
  </Alert>
);
