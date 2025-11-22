import { graphql } from "@/gql";
import { EnvironmentType } from "@/gql/graphql";
import { useEnvironment } from "@/components/Environments/environment-context";

import { useGraphQLQuery } from "@/utils/useGraphQLQuery";
import { Alert } from "@inngest/components/Alert/NewAlert";
import { Button } from "@inngest/components/Button/NewButton";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@inngest/components/DropdownMenu";
import { RiMore2Line, RiPencilLine, RiDeleteBinLine } from "@remixicon/react";
import { createFileRoute, notFound } from "@tanstack/react-router";
import { useState } from "react";

import { Secret, type SecretKind } from "@/components/Secret/Secret";
import EditKeyNameModal from "@/components/Manage/EditKeyNameModal";
import TransformEvent from "@/components/Manage/TransformEvent";
import DeleteKeyModal from "@/components/Manage/DeleteKeyModal";
import FilterEvents from "@/components/Manage/FilterEvents";
import { Provider } from "@/components/Manage/Context";

const GetKeyDocument = graphql(`
  query GetIngestKey($environmentID: ID!, $keyID: ID!) {
    environment: workspace(id: $environmentID) {
      ingestKey(id: $keyID) {
        id
        name
        createdAt
        presharedKey
        url
        filter {
          type
          ips
          events
        }
        metadata
        source
      }
    }
  }
`);

const SOURCE_INTEGRATION = "integration";

function isFilterType(value: string): value is "allow" | "deny" {
  return value === "allow" || value === "deny";
}

export const Route = createFileRoute(
  "/_authed/_org-active/env/$envSlug/manage/$ingestKeys/$keyID/",
)({
  component: KeyComponent,
});

function KeyComponent() {
  const { keyID, ingestKeys } = Route.useParams();

  const [isDeleteKeyModalVisible, setIsDeleteKeyModalVisible] = useState(false);
  const [isEditKeyNameModalVisible, setIsEditKeyNameModalVisible] =
    useState(false);

  const environment = useEnvironment();

  const { data, isLoading, error } = useGraphQLQuery({
    query: GetKeyDocument,
    variables: {
      environmentID: environment.id,
      keyID,
    },
  });

  if (isLoading) {
    return <>{/* To do: skeleton */}</>;
  }

  const key = data?.environment.ingestKey;

  if (error || !key) {
    throw notFound();
  }

  const filterType = key.filter.type;
  if (!filterType || !isFilterType(filterType)) {
    throw new Error(`invalid filter type: ${filterType}`);
  }

  // Integration created keys cannot be deleted or renamed
  const isIntegration = key.source === SOURCE_INTEGRATION;

  let secretKind: SecretKind;
  let value;
  if (ingestKeys === "webhooks") {
    secretKind = "webhook-path";
    value = key.url || "";
  } else {
    secretKind = "event-key";
    value = key.presharedKey;
  }

  return (
    <div>
      {ingestKeys === "keys" &&
        environment.type === EnvironmentType.BranchParent && (
          <Alert
            className="flex items-center rounded-none text-sm"
            severity="info"
          >
            Event keys are shared for all branch environments
          </Alert>
        )}
      <div className="divide-subtle m-6 divide-y">
        <Provider initialState={key}>
          <div className="pb-8">
            <div className="mb-8 flex items-center gap-2">
              <h2 className="text-lg font-semibold">{key.name}</h2>
              {/* TO DO: move this to the Header as ActionsMenu */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    kind="secondary"
                    appearance="outlined"
                    size="medium"
                    icon={<RiMore2Line />}
                  />
                </DropdownMenuTrigger>
                <DropdownMenuContent>
                  <DropdownMenuItem
                    onSelect={() => setIsEditKeyNameModalVisible(true)}
                  >
                    <RiPencilLine className="h-4 w-4" />
                    Edit name
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onSelect={() => setIsDeleteKeyModalVisible(true)}
                    className="text-error"
                  >
                    <RiDeleteBinLine className="h-4 w-4" />
                    Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
              <DeleteKeyModal
                keyID={keyID}
                isOpen={isDeleteKeyModalVisible}
                onClose={() => setIsDeleteKeyModalVisible(false)}
                description={
                  isIntegration
                    ? "Warning: This key was created via integration. Please confirm that you are no longer using it before deleting."
                    : ""
                }
              />
              <EditKeyNameModal
                keyID={keyID}
                keyName={key.name}
                isOpen={isEditKeyNameModalVisible}
                onClose={() => setIsEditKeyNameModalVisible(false)}
              />
              {key.source === SOURCE_INTEGRATION && (
                <span className="text-subtle ml-8 text-sm">
                  Created via integration
                </span>
              )}
            </div>
            <div className="w-3/5">
              <Secret kind={secretKind} secret={value} />
            </div>
          </div>
          <TransformEvent
            keyID={keyID}
            metadata={key.metadata}
            keyName={key.name}
          />
          <FilterEvents
            keyID={keyID}
            filter={{
              ...key.filter,
              type: filterType,
            }}
            keyName={key.name}
          />
        </Provider>
      </div>
    </div>
  );
}
