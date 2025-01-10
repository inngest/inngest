import { NotFoundIcon } from '@/icons/NotFoundIcon';

export const NotFound = ({}) => {
  return (
    <div className="bg-canvasBase border-subtle overflowx-hidden relative flex h-[384px] w-full flex-col rounded-md border p-5">
      <div className="bg-canvasBase flex h-full w-full flex-col items-center justify-center gap-3 overflow-x-hidden rounded-md p-2 text-center md:px-12 ">
        <NotFoundIcon />
        <div className="text-lg font-medium">No data found</div>
        <div className="text-subtle text-sm leading-tight">
          Feel free to explore the filters to view more data.
        </div>
      </div>
    </div>
  );
};
