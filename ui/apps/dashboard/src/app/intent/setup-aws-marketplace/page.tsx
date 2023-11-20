'use client';

import { useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { type AwsMarketplaceSetupInput, type MakeMaybe } from '@/gql/graphql';
import AWSLogo from '@/icons/aws-logo.svg';
import ApprovalDialog from '../ApprovalDialog';

const CompleteAWSMarketplaceSetup = graphql(`
  mutation CompleteAWSMarketplaceSetup($input: AWSMarketplaceSetupInput!) {
    completeAWSMarketplaceSetup(input: $input) {
      message
    }
  }
`);

export default function Page() {
  const params = useSearchParams();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>('');
  const [, completeSetup] = useMutation(CompleteAWSMarketplaceSetup);

  // Params and validation
  const setupParams: AwsMarketplaceSetupInput = useMemo(
    function () {
      return {
        customerID: params.get('customer_id') || '',
        productCode: params.get('product_code') || '',
        awsAccountID: params.get('aws_account_id') || '',
      };
    },
    [params]
  );

  useEffect(() => {
    for (const key in setupParams) {
      if (setupParams[key as keyof typeof setupParams] === '') {
        setError(`Malformed URL: Missing ${key} parameter`);
        continue;
      }
    }
  }, [setupParams]);

  async function approve() {
    setLoading(true);

    completeSetup({
      input: setupParams,
    }).then((result) => {
      setLoading(false);
      if (result.error) {
        setError(result.error.message);
        console.log('error', result.error);
      } else {
        // TODO - Setup complete and give options to proceed to dashboard or close?
        console.log('result', result);
      }
    });
  }

  function cancel() {
    window.close();
  }

  return (
    <ApprovalDialog
      title="Complete AWS Marketplace setup with Inngest"
      description={
        <>
          <p className="my-6">
            This will link your AWS Marketplace subscription to your Inngest account.
          </p>
          <p className="my-6">
            Your account billing will be managed within your AWS account. You can always get support
            from the Inngest team by emailing{' '}
            <a href="mailto:hello@inngest.com" target="_blank">
              hello@inngest.com
            </a>
            .
          </p>
        </>
      }
      graphic={
        <>
          <AWSLogo className="h-16 w-16" />
        </>
      }
      isLoading={loading}
      onApprove={approve}
      onCancel={cancel}
      error={error}
      secondaryInfo={
        <>
          If you currently have an existing billing plan with your Inngest account, <br />
          please cancel it first, or reach out to the Inngest team.
        </>
      }
    />
  );
}
