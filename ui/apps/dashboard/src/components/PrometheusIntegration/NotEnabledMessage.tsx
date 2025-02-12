import { Alert } from '@inngest/components/Alert';

export default function NotEnabledMessage() {
  return (
    <Alert severity="warning">
      <p>Your Inngest plan does not support Prometheus integration.</p>
      <p className="mt-2">
        To use this feature,{' '}
        <Alert.Link size="medium" severity="warning" href="/billing" className="inline underline">
          upgrade your plan
        </Alert.Link>
        .
      </p>
    </Alert>
  );
}
