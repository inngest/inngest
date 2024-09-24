import { Alert } from '@inngest/components/Alert/Alert';
import { NewLink } from '@inngest/components/Link/Link';

//
// A thin wrapper around Alert for a standard error + contact support message
export const Error = ({ message }: { message: string }) => (
  <Alert severity="error" className="mb-4 inline w-full">
    {message} If the problem persists, contact support
    <NewLink
      href="/support"
      className="text-error decoration-error hover:decoration-error ml-1 inline"
    >
      here
    </NewLink>
    .
  </Alert>
);
