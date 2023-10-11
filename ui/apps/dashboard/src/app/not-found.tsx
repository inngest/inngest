import { ExclamationCircleIcon } from '@heroicons/react/20/solid';

export default function KeysNotFound() {
  return (
    <div className="mesh-gradient flex h-full w-full flex-col items-center justify-center">
      <div className="flex flex-row items-center gap-2 rounded-lg bg-slate-50 p-6 text-slate-900 shadow">
        <ExclamationCircleIcon className="h-6 w-6" />
        <h1 className="text-lg">404 - Page Not Found</h1>
      </div>
    </div>
  );
}
