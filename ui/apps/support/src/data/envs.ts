import { queryOptions } from "@tanstack/react-query";
import { createServerFn } from "@tanstack/react-start";
import { getAuthHeaders, inngestGQLAPI } from "./gqlApi";
import { graphql } from "@/gql";

const GetEnvironmentBySlugDocument = graphql(`
  query GetEnvironmentBySlug($slug: String!) {
    envBySlug(slug: $slug) {
      id
      name
      slug
      parentID
      test
      type
      createdAt
      lastDeployedAt
      isArchived
      isAutoArchiveEnabled
      webhookSigningKey
    }
  }
`);

export const envQueryOptions = (slug: string) =>
  queryOptions({
    queryKey: ["envBySlug", slug],
    queryFn: () => fetchEnvBySlug({ data: { slug } }),
  });

export const fetchEnvBySlug = createServerFn({ method: "GET" })
  .inputValidator((data: { slug: string }) => data)
  .handler(async ({ data }) => {
    console.info(`gql api fetching env by slug ${data.slug}...`);
    return await inngestGQLAPI.request(
      GetEnvironmentBySlugDocument,
      {
        slug: data.slug,
      },
      await getAuthHeaders(),
    );
  });
