import { toast } from 'sonner';
import { ulid } from 'ulid';

import Button from '@/components/Button/Button';
import { useSendEventMutation } from '@/store/devApi';

type TriggerCronButtonProps = {
  functionId: string;
  label?: string;
  appearance?: 'solid' | 'outlined' | 'text';
};

export default function TriggerCronButton({
  functionId,
  label = 'Trigger',
  appearance,
}: TriggerCronButtonProps) {
  const [sendEvent] = useSendEventMutation();

  return (
    <Button
      label={label}
      appearance={appearance}
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
          });
      }}
    />
  );
}
