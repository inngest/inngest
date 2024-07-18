import { Select } from '@inngest/components/Select/Select';
import { RiEqualizer2Line } from '@remixicon/react';

export default function KeysMenu({ collapsed }: { collapsed: boolean }) {
  return (
    <Select onChange={(v) => null} isLabelVisible={false} multiple={false}>
      <Select.Button
        isLabelVisible={false}
        className="border-muted h-[18px] w-[18px] rounded"
      ></Select.Button>
    </Select>
  );
}
