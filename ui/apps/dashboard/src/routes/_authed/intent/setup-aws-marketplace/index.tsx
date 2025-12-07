import { useEffect, useMemo, useState } from "react";
import { Link } from "@inngest/components/Link/NewLink";
import { useMutation } from "urql";
import { createFileRoute, useNavigate } from "@tanstack/react-router";

import { graphql } from "@/gql";
import { type AwsMarketplaceSetupInput } from "@/gql/graphql";
import AWSLogo from "@/components/Icons/aws-logo.svg?react";

import { pathCreator } from "@/utils/urls";
import ApprovalDialog from "@/components/Intent/ApprovalDialog";
import { useSearchParam } from "@inngest/components/hooks/useNewSearchParams";

export const Route = createFileRoute("/_authed/intent/setup-aws-marketplace/")({
  component: SetupAWSMarketplacePage,
});

const CompleteAWSMarketplaceSetup = graphql(`
  mutation CompleteAWSMarketplaceSetup($input: AWSMarketplaceSetupInput!) {
    completeAWSMarketplaceSetup(input: $input) {
      message
    }
  }
`);

function SetupAWSMarketplacePage() {
  const navigate = useNavigate();
  const [customerID] = useSearchParam("customer_id");
  const [productCode] = useSearchParam("product_code");
  const [awsAccountID] = useSearchParam("aws_account_id");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | React.ReactNode>("");
  const [, completeSetup] = useMutation(CompleteAWSMarketplaceSetup);

  //
  // Params and validation
  const setupParams: AwsMarketplaceSetupInput = useMemo(
    () => ({
      customerID: customerID || "",
      productCode: productCode || "",
      awsAccountID: awsAccountID || "",
    }),
    [customerID, productCode, awsAccountID],
  );

  useEffect(() => {
    for (const key in setupParams) {
      if (setupParams[key as keyof typeof setupParams] === "") {
        setError(`Malformed URL: Missing ${key} parameter`);
        continue;
      }
    }
  }, [setupParams]);

  const approve = async () => {
    setLoading(true);

    completeSetup({
      input: setupParams,
    }).then((result) => {
      setLoading(false);
      if (result.error) {
        const cleanError = result.error.message.replace("[GraphQL]", "").trim();
        setError(
          <>
            {cleanError}.{" "}
            <Link size="medium" className="inline-flex" href="/support">
              Contact support
            </Link>{" "}
            or{" "}
            <Link
              size="medium"
              className="inline-flex"
              to={pathCreator.billing()}
            >
              manage billing
            </Link>
            .
          </>,
        );
        console.log("error", result.error);
      } else {
        navigate({ to: pathCreator.billing() });
      }
    });
  };

  const cancel = () => {
    if (window.opener != null || window.history.length == 1) {
      window.close();
    } else {
      navigate({ to: "/" });
    }
  };

  return (
    <ApprovalDialog
      title="Complete AWS Marketplace setup with Inngest"
      description={
        <>
          <p className="my-6">
            This will link your AWS Marketplace subscription to your Inngest
            account.
          </p>
          <p className="my-6">
            Your account billing will be managed within your AWS account. You
            can always get support from the Inngest team by emailing{" "}
            <a href="mailto:hello@inngest.com" target="_blank">
              hello@inngest.com
            </a>
            .
          </p>
        </>
      }
      graphic={<AWSLogo className="h-16 w-16" />}
      isLoading={loading}
      onApprove={approve}
      onCancel={cancel}
      error={error}
      secondaryInfo={
        <>
          If you currently have an existing billing plan with your Inngest
          account, <br />
          please cancel it first, or reach out to the Inngest team.
        </>
      }
    />
  );
}
