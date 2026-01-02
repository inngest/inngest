import { queryOptions } from "@tanstack/react-query";
import { createServerFn } from "@tanstack/react-start";
import { auth, inngestGQLAPI } from "./gqlApi";
import type { GetEnvironmentBySlugQuery } from "@/gql/graphql";
import { GetEnvironmentBySlugDocument } from "@/gql/graphql";

export const envQueryOptions = (slug: string) =>
  queryOptions({
    queryKey: ["envBySlug", slug],
    queryFn: () => fetchEnvBySlug({ data: { slug } }),
  });

export const fetchEnvBySlug = createServerFn({ method: "GET" })
  .inputValidator((data: { slug: string }) => data)
  .handler(async ({ data }) => {
    console.info(`gql api fetching env by slug ${data.slug}...`);
    return await inngestGQLAPI.request<GetEnvironmentBySlugQuery>(
      GetEnvironmentBySlugDocument,
      {
        slug: data.slug,
      },
      await auth(),
    );
  });
