import type { TicketStatusFilter } from "@/data/plain";

type FiltersProps = {
  status: TicketStatusFilter | undefined;
  onStatusChange: (status: TicketStatusFilter | undefined) => void;
};

const options: Array<{ label: string; value: TicketStatusFilter | undefined }> =
  [
    { label: "All", value: undefined },
    { label: "Open", value: "open" },
    { label: "Closed", value: "closed" },
  ];

export function Filters({ status, onStatusChange }: FiltersProps) {
  return (
    <div className="flex w-full items-center gap-1 text-sm">
      {options.map((option) => {
        const isActive = status === option.value;
        return (
          <button
            key={option.label}
            type="button"
            onClick={() => onStatusChange(option.value)}
            className={`rounded px-2.5 py-1 text-sm font-medium transition-colors ${
              isActive
                ? "bg-contrast text-onContrast"
                : "text-muted hover:text-basis hover:bg-canvasSubtle"
            }`}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}
