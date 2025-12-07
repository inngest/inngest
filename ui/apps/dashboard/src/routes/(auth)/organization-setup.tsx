import ReloadClerkAndRedirect from "@/components/Clerk/ReloadClerkAndRedirect";
import { graphql } from "@/gql";
import graphqlAPI from "@/queries/graphqlAPI";
import { pathCreator } from "@/utils/urls";
import { createFileRoute, redirect } from "@tanstack/react-router";
import { createServerFn } from "@tanstack/react-start";

const SetUpAccountDocument = graphql(`
  mutation SetUpAccount {
    setUpAccount {
      account {
        id
      }
    }
  }
`);

const setUpAccount = createServerFn({ method: "GET" }).handler(() =>
  graphqlAPI.request(SetUpAccountDocument),
);

export const Route = createFileRoute("/(auth)/organization-setup")({
  component: OrganizationSetupComponent,
  loader: async () => {
    await setUpAccount();
  },
});

function OrganizationSetupComponent() {
  return <ReloadClerkAndRedirect redirectURL={pathCreator.onboarding()} />;
}
