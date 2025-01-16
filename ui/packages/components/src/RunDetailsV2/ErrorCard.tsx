import { Alert } from '@inngest/components/Alert';
import { Button } from '@inngest/components/Button';

type Props = {
  error: Error | any;
  reset?: () => void;
};

export function ErrorCard({ error, reset }: Props) {
  return (
    <Alert
      severity="error"
      button={
        reset && (
          <Button onClick={() => reset()} kind="secondary" appearance="outlined" label="Reload" />
        )
      }
    >
      <p className="mb-4 font-semibold">{error.message}</p>
      <p>
        An error occurred loading. Click reload to try again. If the problem persists, contact
        support.
      </p>
    </Alert>
  );
}
