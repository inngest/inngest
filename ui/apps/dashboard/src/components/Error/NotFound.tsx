import { RiErrorWarningLine } from '@remixicon/react';

export default function NotFound() {
  return (
    <div className="flex flex-row items-center justify-center h-full w-full gap-2">
      <RiErrorWarningLine className="h-6 w-6" />
      <h1 className="text-lg">404 - Page Not Found</h1>
    </div>
  );
}
