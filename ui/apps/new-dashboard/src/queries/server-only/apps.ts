import { GetProductionAppsDocument } from "@/components/Onboarding/data";
import {
  GetProductionWorkspaceDocument,
  type ProductionAppsQuery,
} from "@/gql/graphql";
import { workspacesToEnvironments } from "@/utils/environments";
import { createServerFn } from "@tanstack/react-start";
import graphqlAPI from "../graphqlAPI";

export const getProdApps = createServerFn({
  method: "GET",
}).handler(async () => {
  const query = await graphqlAPI.request(GetProductionWorkspaceDocument);
  const environment = workspacesToEnvironments([query.defaultEnv])[0];

  return await graphqlAPI.request<ProductionAppsQuery>(
    GetProductionAppsDocument,
    {
      envID: environment.id,
    },
  );
});
