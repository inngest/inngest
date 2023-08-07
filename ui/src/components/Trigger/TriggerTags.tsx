import { FunctionTriggerTypes } from '@/store/generated';
import TriggerTag from './TriggerTag';

type TriggerTagsProps = {
  triggers: {
    type: FunctionTriggerTypes.Event | FunctionTriggerTypes.Cron;
    value: string;
  }[];
};

export default function TriggerTags({ triggers }: TriggerTagsProps) {
  return (
    <>
      {triggers?.map((trigger, index) => {
        return <TriggerTag key={index} value={trigger.value} type={trigger.type} />;
      })}
    </>
  );
}
