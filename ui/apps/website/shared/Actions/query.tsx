import { useQuery } from "urql";
import { ActionCategory } from "./types";
import { Action } from "src/types";

const bothGQL = `
query Actions {
  actions {
    dsn
    tagline
    category { name }
    latest {
      name
      dsn
      versionMajor
      versionMinor
      Settings
      WorkflowMetadata {
        name expression required type form
      }
      Response
      Edges { type name if async { match event ttl } }
    }
  }
}
`;

export const useActionCategories = (pause: boolean) => {
  return useQuery<{ categories: ActionCategory[] }>({ query: bothGQL, pause });
};

export const useActionsWithCategory = (pause: boolean) => {
  return useQuery<{ actions: Action[] }>({ query: bothGQL, pause });
};
