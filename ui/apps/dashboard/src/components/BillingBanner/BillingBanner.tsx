import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { BillingBannerView } from './BillingBannerView';

export async function BillingBanner() {
  let entitlementUsage;
  try {
    entitlementUsage = (await graphqlAPI.request(entitlementUsageQuery)).account.entitlementUsage;
  } catch (e) {
    console.error(e);
    return null;
  }

  return <BillingBannerView entitlementUsage={entitlementUsage} />;
}

const entitlementUsageQuery = graphql(`
  query EntitlementUsage {
    account {
      id
      entitlementUsage {
        runCount {
          current
          limit
        }
      }
    }
  }
`);
