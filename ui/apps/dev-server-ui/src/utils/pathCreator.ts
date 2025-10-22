export const pathCreator = {
  app({ externalAppID }: { externalAppID: string }): string {
    // TODO: Make this goes to a specific app page when we add that feature
    return '/apps'
  },
  eventPopout({ eventID }: { eventID: string }): string {
    return `/event?eventID=${eventID}`
  },
  function({ functionSlug }: { functionSlug: string }): string {
    // TODO: Make this goes to a specific app page when we add that feature
    return '/functions'
  },
  runPopout({ runID }: { runID: string }): string {
    return `/run?runID=${runID}`
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
  }): string {
    const params = new URLSearchParams()
    params.set('function', functionSlug)
    runID && params.set('runID', runID)
    debugRunID && params.set('debugRunID', debugRunID)
    debugSessionID && params.set('debugSessionID', debugSessionID)

    return `/debugger/function?${params.toString()}`
  },
}
