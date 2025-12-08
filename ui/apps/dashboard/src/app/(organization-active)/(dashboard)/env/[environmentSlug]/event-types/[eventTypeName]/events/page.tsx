import EventsPage from "@/components/Events/EventsPage";

export default function Page({
  params: { environmentSlug: envSlug, eventTypeName },
}: {
  params: { environmentSlug: string; eventTypeName: string };
}) {
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
