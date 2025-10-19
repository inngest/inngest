import { useCallback } from 'react'

import { client } from '@/store/baseApi'
import {
  GetEventV2Document,
  GetEventV2PayloadDocument,
  GetEventV2RunsDocument,
  GetEventsV2Document,
  type GetEventV2PayloadQuery,
  type GetEventV2PayloadQueryVariables,
  type GetEventV2Query,
  type GetEventV2QueryVariables,
  type GetEventV2RunsQuery,
  type GetEventV2RunsQueryVariables,
  type GetEventsV2Query,
  type GetEventsV2QueryVariables,
} from '@/store/generated'

export function useEvents() {
  return useCallback(
    async ({
      cursor,
      endTime,
      // source,
      eventNames,
      startTime,
      celQuery,
      includeInternalEvents,
    }: GetEventsV2QueryVariables) => {
      const data: GetEventsV2Query = await client.request(GetEventsV2Document, {
        cursor,
        endTime,
        // source,
        eventNames,
        startTime,
        celQuery,
        includeInternalEvents,
      })
      const eventsData = data.eventsV2
      const events = eventsData.edges.map(({ node }) => ({
        ...node,
        receivedAt: new Date(node.receivedAt),
        runs: node.runs.map((run) => ({
          fnName: run.function.name,
          fnSlug: run.function.slug,
          status: run.status,
          id: run.id,
          completedAt: run.endedAt ? new Date(run.endedAt) : undefined,
          startedAt: run.startedAt ? new Date(run.startedAt) : undefined,
        })),
      }))

      return {
        events,
        pageInfo: eventsData.pageInfo,
        totalCount: eventsData.totalCount,
      }
    },
    [],
  )
}

export function useEventDetails() {
  return useCallback(async ({ eventID }: GetEventV2QueryVariables) => {
    const data: GetEventV2Query = await client.request(GetEventV2Document, {
      eventID,
    })
    const eventData = data.eventV2
    return {
      ...eventData,
      receivedAt: new Date(eventData.receivedAt),
      occurredAt: eventData.occurredAt
        ? new Date(eventData.occurredAt)
        : undefined,
    }
  }, [])
}

export function useEventPayload() {
  return useCallback(async ({ eventID }: GetEventV2PayloadQueryVariables) => {
    const data: GetEventV2PayloadQuery = await client.request(
      GetEventV2PayloadDocument,
      {
        eventID,
      },
    )

    const eventData = data.eventV2.raw

    return { payload: eventData }
  }, [])
}

export function useEventRuns() {
  return useCallback(async ({ eventID }: GetEventV2RunsQueryVariables) => {
    const data: GetEventV2RunsQuery = await client.request(
      GetEventV2RunsDocument,
      {
        eventID,
      },
    )

    const eventData = data.eventV2
    return {
      ...eventData,
      runs: eventData.runs.map((run) => ({
        fnName: run.function.name,
        fnSlug: run.function.slug,
        status: run.status,
        id: run.id,
        completedAt: run.endedAt ? new Date(run.endedAt) : undefined,
        startedAt: run.startedAt ? new Date(run.startedAt) : undefined,
      })),
    }
  }, [])
}
