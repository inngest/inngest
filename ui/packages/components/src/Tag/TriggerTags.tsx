import { TriggerTypes } from '@inngest/components/types/triggers';

import { TriggerTag } from './TriggerTag';

type TriggerTagsProps = {
  triggers: {
    type: TriggerTypes;
    value: string;
  }[];
};

export function TriggerTags({ triggers }: TriggerTagsProps) {
  return (
    <>
      {triggers?.map((trigger, index) => {
        return <TriggerTag key={index} value={trigger.value} type={trigger.type} />;
      })}
    </>
  );
}
