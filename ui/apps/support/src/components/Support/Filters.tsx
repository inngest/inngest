import { RiArrowDownSLine } from "@remixicon/react";

type FiltersProps = {
  onStatusChange?: (status: string) => void;
  selectedStatus?: string;
};

export function Filters(_props: FiltersProps) {
  return (
    <div className="flex w-full items-start px-4 py-2 text-sm">
      <div className="border-muted bg-canvasBase flex items-center overflow-clip rounded border-[0.75px]">
        <div className="border-muted bg-canvasBase flex items-center gap-2 rounded border px-2 py-1.5">
          <div className="flex items-center gap-1">
            <div className="text-basis flex flex-col justify-center overflow-ellipsis overflow-hidden leading-none">
              <p className="overflow-ellipsis overflow-hidden leading-4">
                Status
              </p>
            </div>
            <div className="relative h-4 w-4">
              <div className="absolute left-0 top-0 h-4 w-4 overflow-clip">
                <RiArrowDownSLine className="text-muted h-4 w-4" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
