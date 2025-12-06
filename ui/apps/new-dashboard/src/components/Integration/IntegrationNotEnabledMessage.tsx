import { Alert } from "@inngest/components/Alert/NewAlert";

type Props = {
  integrationName: string;
};

export default function IntegrationNotEnabledMessage({
  integrationName,
}: Props) {
  return (
    <Alert severity="warning">
      <p>Your Inngest plan does not support {integrationName} integration.</p>
      <p className="mt-2">
        To use this feature,{" "}
        <Alert.Link
          severity="warning"
          href="/billing"
          className="inline underline"
        >
          upgrade your plan
        </Alert.Link>
        .
      </p>
    </Alert>
  );
}
