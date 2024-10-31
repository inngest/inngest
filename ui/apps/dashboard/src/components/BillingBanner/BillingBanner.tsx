import { NewButton } from '@inngest/components/Button';

import { graphql } from '@/gql';
import graphqlAPI from '@/queries/graphqlAPI';
import { pathCreator } from '@/utils/urls';
import { Banner } from '../Banner';
import { parseEntitlementUsage } from './parse';

export async function BillingBanner() {
  let res;
  try {
    res = await graphqlAPI.request(query);
  } catch (e) {
    console.error(e);
    return null;
  }

  const { bannerMessage, bannerSeverity, items } = parseEntitlementUsage(
    res.account.entitlementUsage
  );

  return (
    <Banner className="flex" kind={bannerSeverity}>
      <div className="flex grow">
        <div className="grow">
          {bannerMessage}
          <ul className="list-none">
            {items.map(([k, v]) => (
              <li key={k}>{v}</li>
            ))}
          </ul>
        </div>

        <div className="flex items-center">
          <NewButton
            appearance="outlined"
            href={pathCreator.billing()}
            kind="secondary"
            label="Upgrade plan"
          />
        </div>
      </div>
    </Banner>
  );
}

const query = graphql(`
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
