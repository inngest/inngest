import { IntegrationSteps } from '@inngest/components/PostgresIntegrations/types';

export default function NeonStep({ params: { step } }: { params: { step: string } }) {
  if (step === IntegrationSteps.Authorize) {
    return <div>Page for Auth</div>;
  } else if (step === IntegrationSteps.FormatWal) {
    return <div>Page for Format</div>;
  } else if (step === IntegrationSteps.ConnectDb) {
    return <div>Page For Connect</div>;
  }

  return <div>Page Content</div>;
}
