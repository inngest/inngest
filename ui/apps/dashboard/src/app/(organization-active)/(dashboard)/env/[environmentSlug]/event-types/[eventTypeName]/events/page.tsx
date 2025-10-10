import EventsPage from '@/components/Events/EventsPage';

export default async function Page(props: {
  params: Promise<{ environmentSlug: string; eventTypeName: string }>;
}) {
  const params = await props.params;

  const { environmentSlug: envSlug, eventTypeName } = params;

  const decodedEventTypeName = decodeURIComponent(eventTypeName);
  return (
    <>
      <EventsPage
        environmentSlug={envSlug}
        eventTypeNames={[decodedEventTypeName]}
        singleEventTypePage
        showHeader={false}
      />
    </>
  );
}
