type FilterResultDetailsProps = {
  denominator: number;
  numerator: number;
};

export function FilterResultDetails({ denominator, numerator }: FilterResultDetailsProps) {
  return (
    <div className="flex items-center px-3 py-2 max-[500px]:hidden">
      <span className="text-light whitespace-nowrap text-sm">
        Displaying {numerator} of {denominator}
      </span>
    </div>
  );
}
