import { useEnvironment } from "@/components/Environments/environment-context";
import { ReplayList } from "@/components/Functions/ReplayList";
import NewReplayButton from "@/components/Replay/NewReplayButton";
import { graphql } from "@/gql";

import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "urql";

const GetFunctionPauseStateDocument = graphql(`
  query GetFunctionPauseState($environmentID: ID!, $functionSlug: String!) {
    environment: workspace(id: $environmentID) {
      function: workflowBySlug(slug: $functionSlug) {
        id
        isPaused
      }
    }
  }
`);

export const Route = createFileRoute(
  "/_authed/_org-active/env/$envSlug/functions/$slug/replays/",
)({
  component: RouteComponent,
});

function RouteComponent() {
  const { slug } = Route.useParams();
  const env = useEnvironment();
  const functionSlug = decodeURIComponent(slug);
  const [{ data }] = useQuery({
    query: GetFunctionPauseStateDocument,
    variables: {
      environmentID: env.id,
      functionSlug,
    },
  });
  const functionIsPaused = data?.environment.function?.isPaused || false;

  return (
    <>
      {!env.isArchived && !functionIsPaused && (
        <div className="flex items-center justify-end px-5">
          <NewReplayButton functionSlug={functionSlug} />
        </div>
      )}
      <div className="h-full overflow-y-auto">
        <ReplayList functionSlug={functionSlug} />
      </div>
    </>
  );
}
