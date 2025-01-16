'use client';

import { useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Link } from '@inngest/components/Link';
import { useMutation } from 'urql';

import { graphql } from '@/gql';
import { type AwsMarketplaceSetupInput } from '@/gql/graphql';
import AWSLogo from '@/icons/aws-logo.svg';
import { pathCreator } from '@/utils/urls';
import ApprovalDialog from '../ApprovalDialog';

const CompleteAWSMarketplaceSetup = graphql(`
  mutation CompleteAWSMarketplaceSetup($input: AWSMarketplaceSetupInput!) {
    completeAWSMarketplaceSetup(input: $input) {
      message
    }
  }
`);

export default function Page() {
  const router = useRouter();
  const params = useSearchParams();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | React.ReactNode>('');
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
        const cleanError = result.error.message.replace('[GraphQL]', '').trim();
        setError(
          <>
            {cleanError}.{' '}
            <Link size="medium" className="inline-flex" href="/support">
              Contact support
            </Link>{' '}
            or{' '}
            <Link size="medium" className="inline-flex" href={pathCreator.billing()}>
              manage billing
            </Link>
            .
          </>
        );
        console.log('error', result.error);
      } else {
        router.push(pathCreator.billing());
      }
    });
  }

  function cancel() {
    if (window.opener != null || window.history.length == 1) {
      window.close();
    } else {
      router.push('/');
    }
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
