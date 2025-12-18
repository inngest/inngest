import { Alert } from '@inngest/components/Alert/Alert';

import { cn } from '../utils/classNames';

//
// A thin wrapper around Alert for a standard error + contact support message
export const Error = ({
  message,
  button,
  className,
}: {
  message: string;
  button?: React.ReactNode;
  className?: string;
}) => (
  <Alert severity="error" className={cn('mb-4 w-full', className)} button={button}>
    {message} If the problem persists,
    <Alert.Link severity="error" href="/support" size="medium" className="ml-1 inline">
      contact support
    </Alert.Link>
    .
  </Alert>
);
