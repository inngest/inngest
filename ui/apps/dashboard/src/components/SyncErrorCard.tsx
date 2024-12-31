import { Alert } from '@inngest/components/Alert/Alert';

type Props = {
  className?: string;
  error: string;
};

export function SyncErrorCard({ className, error }: Props) {
  return (
    <div className={className}>
      <Alert severity="error">{error}</Alert>
    </div>
  );
}
