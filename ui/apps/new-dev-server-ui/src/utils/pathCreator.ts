import type { Route } from 'next'

export const pathCreator = {
  app({ externalAppID }: { externalAppID: string }): Route {
    // TODO: Make this goes to a specific app page when we add that feature
    return '/apps' as Route
  },
  eventPopout({ eventID }: { eventID: string }): Route {
    return `/event?eventID=${eventID}` as Route
  },
  function({ functionSlug }: { functionSlug: string }): Route {
    // TODO: Make this goes to a specific app page when we add that feature
    return '/functions' as Route
  },
  runPopout({ runID }: { runID: string }): Route {
    return `/run?runID=${runID}` as Route
  },
  debugger({
    functionSlug,
    runID,
    debugRunID,
    debugSessionID,
  }: {
    functionSlug: string
    runID?: string
    debugRunID?: string | null
    debugSessionID?: string | null
  }): Route {
    const params = new URLSearchParams()
    params.set('function', functionSlug)
    runID && params.set('runID', runID)
    debugRunID && params.set('debugRunID', debugRunID)
    debugSessionID && params.set('debugSessionID', debugSessionID)

    return `/debugger/function?${params.toString()}` as Route
  },
}
