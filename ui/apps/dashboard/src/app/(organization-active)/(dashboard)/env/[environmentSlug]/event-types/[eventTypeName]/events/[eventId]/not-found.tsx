import { Alert } from '@inngest/components/Alert/Alert';

export default function EventNotFound() {
  return (
    <Alert severity="warning">
      <p className="text-sm">Could not find the requested event</p>
    </Alert>
  );
}
