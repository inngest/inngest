import { toast } from 'sonner';
import { ulid } from 'ulid';

import Button from '@/components/Button';
import { useSendEventMutation } from '@/store/devApi';
import { selectEvent } from '@/store/global';
import { useAppDispatch } from '@/store/hooks';

type TriggerCronButtonProps = {
  functionId: string;
  label?: string;
  kind?: 'primary' | 'secondary' | 'text';
};

export default function TriggerCronButton({
  functionId,
  label = 'Trigger',
  kind,
}: TriggerCronButtonProps) {
  const [sendEvent] = useSendEventMutation();
  const dispatch = useAppDispatch();

  return (
    <Button
      label={label}
      kind={kind}
      btnAction={() => {
        const id = ulid();

        sendEvent({
          id,
          name: '',
          ts: Date.now(),
          functionId,
        })
          .unwrap()
          .then(() => {
            toast.success('Cron triggered successfully');
            dispatch(selectEvent(id));
          });
      }}
    />
  );
}
