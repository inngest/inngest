"use client";

type FilterResultDetailsProps = {
  size: number;
};

export function FilterResultDetails({ size }: FilterResultDetailsProps) {
  return (
    <div className="flex items-center px-3 py-2 max-[625px]:hidden">
      <span className="text-light whitespace-nowrap text-sm">
        {size} {size === 1 ? "Environment" : "Environments"}
      </span>
    </div>
  );
}
