import { BlankSlate } from '@/components/Blank';

export default function FeedEvents() {
  return (
    <div className="flex-1">
      <BlankSlate
        title="No event selected"
        subtitle="Select an event from the stream on the left to view its details and which functions it's triggered."
        imageUrl="/images/no-fn-selected.png"
      />
    </div>
  );
}
