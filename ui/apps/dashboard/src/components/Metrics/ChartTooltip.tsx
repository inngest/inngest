import type { DefaultLabelFormatterCallbackParams } from '@inngest/components/Chart/Chart.jsx';
import { RiFileCopyLine } from '@remixicon/react';

export const ChartTooltip = ({
  seriesName,
  value,
  name,
  color,
}: DefaultLabelFormatterCallbackParams) => {
  return (
    <div
      style={{ maxWidth: '300px;', borderColor: String(color) }}
      className={` flex flex-col justify-start gap-1 border-l-4 py-2`}
    >
      <div className="text-subtle overflow-hidden text-ellipsis px-3 pt-1 text-xs font-medium uppercase tracking-wide">
        {seriesName}
      </div>
      <div className="text-basis px-3 font-normal leading-normal">{String(value)}</div>
      <hr />
      <div className="text-subtle flex flex-row items-center justify-start gap-2 px-3 text-xs leading-normal">
        <RiFileCopyLine className="h-3 w-3 cursor-pointer" />
        <div>{name}</div>
      </div>
    </div>
  );
};
