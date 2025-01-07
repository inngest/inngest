import { Alert } from '@inngest/components/Alert/Alert';

//
// A thin wrapper around Alert for a standard error + contact support message
export const Error = ({ message, button }: { message: string; button?: React.ReactNode }) => (
  <Alert severity="error" className="mb-4 w-full" button={button}>
    {message} If the problem persists,
    <Alert.Link severity="error" href="/support" size="medium" className="ml-1 inline">
      contact support
    </Alert.Link>
    .
  </Alert>
);
