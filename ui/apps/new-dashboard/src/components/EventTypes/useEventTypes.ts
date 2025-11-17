import { useCallback } from "react";
import { getTimestampDaysAgo } from "@inngest/components/utils/date";
import { useQuery } from "@tanstack/react-query";
import { useClient } from "urql";

import { useEnvironment } from "@/components/Environments/environment-context";
import {
  GetEventTypesV2Document,
  GetEventTypeVolumeV2Document,
  GetEventTypeDocument,
  GetAllEventNamesDocument,
} from "@/gql/graphql";

type QueryVariables = {
  archived: boolean;
  nameSearch: string | null;
  cursor: string | null;
};

export function useEventTypes() {
  const envID = useEnvironment().id;
  const client = useClient();
  return useCallback(
    async ({ cursor, archived, nameSearch }: QueryVariables) => {
      const result = await client
        .query(
          GetEventTypesV2Document,
          {
            envID,
            archived,
            cursor,
            nameSearch,
          },
          { requestPolicy: "network-only" },
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error("no data returned");
      }

      const eventTypesData = result.data.environment.eventTypesV2;
      const events = eventTypesData.edges.map(({ node }) => ({
        name: node.name,
        latestSchema: "",
        functions: node.functions.edges.map((f) => f.node),
        archived,
      }));

      return {
        events,
        pageInfo: eventTypesData.pageInfo,
      };
    },
    [client, envID],
  );
}

type VolumeQueryVariables = {
  eventName: string;
};

export function useEventTypeVolume() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(
    async ({ eventName }: VolumeQueryVariables) => {
      const startTime = getTimestampDaysAgo({
        currentDate: new Date(),
        days: 1,
      }).toISOString();
      const endTime = new Date().toISOString();
      const result = await client
        .query(
          GetEventTypeVolumeV2Document,
          {
            envID,
            eventName,
            startTime,
            endTime,
          },
          { requestPolicy: "network-only" },
        )
        .toPromise();

      if (result.error) {
        throw new Error(result.error.message);
      }

      if (!result.data) {
        throw new Error("no data returned");
      }

      const eventType = result.data.environment.eventType;

      const dailyVolumeSlots = eventType.usage.data.map((slot) => ({
        startCount: slot.count,
        slot: slot.slot,
      }));

      return {
        name: eventType.name,
        volume: {
          totalVolume: eventType.usage.total,
          dailyVolumeSlots,
        },
      };
    },
    [client, envID],
  );
}

export function useEventType({ eventName }: { eventName: string }) {
  const envID = useEnvironment().id;
  const client = useClient();

  return useQuery({
    queryKey: ["event-type", envID, eventName],
    queryFn: async () => {
      const result = await client
        .query(GetEventTypeDocument, { envID, eventName })
        .toPromise();

      if (result.error) {
        throw result.error;
      }

      const eventType = result.data?.environment.eventType;

      if (!eventType) {
        return null;
      }

      return {
        ...eventType,
        functions: eventType.functions.edges.map(({ node }) => node),
      };
    },
  });
}

export function useAllEventTypes() {
  const envID = useEnvironment().id;
  const client = useClient();

  return useCallback(async () => {
    const result = await client
      .query(
        GetAllEventNamesDocument,
        { envID },
        { requestPolicy: "network-only" },
      )
      .toPromise();

    if (result.error) {
      throw new Error(result.error.message);
    }

    if (!result.data) {
      throw new Error("no data returned");
    }

    const eventsData = result.data.environment.eventTypesV2;
    const events = eventsData.edges.map(({ node }) => ({
      id: node.name,
      name: node.name,
      latestSchema: "",
    }));

    return events;
  }, [client, envID]);
}
