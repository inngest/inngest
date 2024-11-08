import ReloadClerkAndRedirect from '@/app/(auth)/ReloadClerkAndRedirect';
import { getBooleanFlag } from '@/components/FeatureFlags/ServerFeatureFlag';
import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { pathCreator } from '@/utils/urls';

const SetUpAccountDocument = graphql(`
  mutation SetUpAccount {
    setUpAccount {
      account {
        id
      }
    }
  }
`);

export default async function OrganizationSetupPage() {
  const onboardingFlow = await getBooleanFlag('onboarding-flow-cloud');
  await graphqlAPI.request(SetUpAccountDocument);
  return (
    <ReloadClerkAndRedirect
      redirectURL={
        onboardingFlow ? pathCreator.onboarding() : pathCreator.apps({ envSlug: 'production' })
      }
    />
  );
}
