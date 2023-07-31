import SlideOver from '@/components/SlideOver';
import { EventSection } from '@/components/Event/Section';
import { FunctionRunSection } from '@/components/Function/RunSection';
import { useAppSelector } from '@/store/hooks';

type StreamDetailsSlideOutProps = {
  isOpen: boolean;
  onClose: () => void;
};

export default function StreamDetailsSlideOut({
  isOpen,
  onClose,
}: StreamDetailsSlideOutProps) {
  const selectedEvent = useAppSelector((state) => state.global.selectedEvent);
  const selectedRun = useAppSelector((state) => state.global.selectedRun);
  if (!selectedEvent) return <></>;
  return (
    <SlideOver isOpen={isOpen} onClose={onClose}>
      <div className="grid grid-cols-2 h-full text-white">
        <EventSection eventId={selectedEvent} />
        <FunctionRunSection runId={selectedRun} />
      </div>
    </SlideOver>
  );
}
