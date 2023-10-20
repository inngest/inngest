import { type Trigger } from '../types/trigger';
import { TriggerTag } from './TriggerTag';

export function TriggerTags({ triggers }: { triggers: Trigger[] }) {
  return (
    <>
      {triggers?.map((trigger, index) => {
        return <TriggerTag key={index} value={trigger.value} type={trigger.type} />;
      })}
    </>
  );
}
