import { type Trigger } from '../types/trigger';
import { TriggerPill } from './TriggerPill';

export function TriggerPills({ triggers }: { triggers: Trigger[] }) {
  return (
    <>
      {triggers?.map((trigger, index) => {
        return <TriggerPill key={index} value={trigger.value} type={trigger.type} />;
      })}
    </>
  );
}
