import { IconSpinner } from '../icons/Spinner';

export function LoadingMore() {
  return (
    <div className="relative mx-auto h-20 w-[510px] overflow-hidden placeholder:mt-4">
      <div
        style={{ borderRadius: '50% / 100% 100% 0 0' }}
        className="absolute top-3 h-24 w-[510px] bg-slate-700/30"
      ></div>
      <IconSpinner className="absolute left-0 right-0 top-10 mx-auto h-6 w-6 fill-white" />
    </div>
  );
}
