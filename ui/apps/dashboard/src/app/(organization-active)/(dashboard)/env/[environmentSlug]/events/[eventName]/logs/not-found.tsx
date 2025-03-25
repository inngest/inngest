import { Alert } from '@inngest/components/Alert/Alert';

export default function EventLogsNotFound() {
  return (
    <Alert severity="warning">
      <p className="text-sm">Could not find any logs for event</p>
    </Alert>
  );
}
