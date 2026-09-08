export const pathCreator = {
  app({ externalAppID }: { externalAppID: string }): string {
    // TODO: Make this goes to a specific app page when we add that feature
    return '/apps';
  },
  eventPopout({ eventID }: { eventID: string }): string {
    return `/event?eventID=${eventID}`;
  },
  function({ functionSlug }: { functionSlug: string }): string {
    const params = new URLSearchParams({ slug: functionSlug });
    return `/functions/config?${params.toString()}`;
  },
  runPopout({ runID }: { runID: string }): string {
    return `/run?runID=${runID}`;
  },
};
