const MAX_HOUR_AM = 12;
const MAX_HOUR_24 = 24;

type TimeInputProps = {
  is24Format?: boolean;
};

export function TimeInput({ is24Format = false }: TimeInputProps) {
  const handlePeriodChange = (e: React.ChangeEvent<HTMLInputElement>) => {};

  return (
    <div className="flex h-8 items-center rounded-lg border-2 border-transparent bg-white px-3.5 text-sm leading-none placeholder-slate-500 transition-all has-[:focus]:border-indigo-500">
      <input
        className="w-7 px-0.5 text-center focus:outline-none"
        placeholder="HH"
        aria-label="Point in time (Hours)"
        min={0}
        max={is24Format ? MAX_HOUR_24 : MAX_HOUR_AM}
        maxLength={2}
      />
      <span className="px-0.5">:</span>
      <input
        className="w-7 px-0.5 text-center focus:outline-none"
        placeholder="mm"
        aria-label="Point in time (Minutes)"
        min={0}
        max={59}
        maxLength={2}
      />
      <span className="px-0.5">:</span>
      <input
        className="w-7 px-0.5 text-center focus:outline-none"
        placeholder="ss"
        aria-label="Point in time (Seconds)"
        min={0}
        max={59}
        maxLength={2}
      />
      <span className="px-0.5">.</span>
      <input
        className="w-9 px-0.5 text-center focus:outline-none"
        placeholder="sss"
        aria-label="Point in time (Milliseconds)"
        min={0}
        max={999}
        maxLength={3}
      />
      {!is24Format && (
        <input
          className="w-7 pl-0.5 focus:outline-none"
          placeholder="AM"
          aria-label="Point in time (Period)"
          maxLength={2}
          onChange={handlePeriodChange}
        />
      )}
    </div>
  );
}
