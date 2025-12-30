import { envQueryOptions } from "@/data/envs";
import { getProfileDisplay } from "@/data/profile";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useServerFn } from "@tanstack/react-start";
import { Header } from "@inngest/components/Header/Header";

export const Route = createFileRoute("/_authed/support/")({
  component: HomeComponent,
  loader: async ({ context }) => {
    const envs = await context.queryClient.ensureQueryData(
      envQueryOptions("production"),
    );

    return {
      envs,
    };
  },
});

function HomeComponent() {
  const { envs } = Route.useLoaderData();
  const getProfile = useServerFn(getProfileDisplay);

  const { data } = useQuery({
    queryKey: ["profile"],
    queryFn: () => getProfile(),
  });

  return (
    <>
      <Header breadcrumb={[{ text: "Support" }]} />
      <div className="m-8 flex flex-col gap-2">
        Example server side data fetch: {envs.envBySlug?.name}
        <div>Example client side data fetch: {data?.displayName}</div>
      </div>
    </>
  );
}
