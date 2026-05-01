import { graphql } from '@/gql';
import { type ActiveBannersQuery } from '@/gql/graphql';
import graphqlAPI from '@/queries/graphqlAPI';
import { createServerFn } from '@tanstack/react-start';

export const activeBannersDocument = graphql(`
  query ActiveBanners {
    account {
      id
      activeBanners {
        id
        severity
        title
        body
        cta {
          label
          url
        }
        dismissible
      }
    }
  }
`);

export const activeBanners = createServerFn({
  method: 'GET',
}).handler(async () => {
  const res = await graphqlAPI.request<ActiveBannersQuery>(
    activeBannersDocument,
  );
  return res.account.activeBanners;
});
