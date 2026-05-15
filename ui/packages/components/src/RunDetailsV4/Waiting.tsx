export const Waiting = () => (
  <div className="flex items-center justify-start">
    <span className="relative ml-4 flex h-2.5 w-2.5">
      <span className="bg-status-queued absolute inline-flex h-full w-full animate-ping rounded-full opacity-75"></span>
      <span className="bg-status-queued relative inline-flex h-2.5 w-2.5 rounded-full"></span>
    </span>
    <p className="text-subtle max-h-24 text-ellipsis break-words py-2.5 pl-3 text-sm">
      Queued run awaiting start...
    </p>
  </div>
);
