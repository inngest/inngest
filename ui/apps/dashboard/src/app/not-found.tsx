import { RiErrorWarningLine } from '@remixicon/react';

export default function KeysNotFound() {
  return (
    <div className="mesh-gradient flex h-full w-full flex-col items-center justify-center">
      <div className="bg-canvasSubtle flex flex-row items-center gap-2 rounded-md p-6 shadow">
        <RiErrorWarningLine className="h-6 w-6" />
        <h1 className="text-lg">404 - Page Not Found</h1>
      </div>
    </div>
  );
}
