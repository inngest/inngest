import { RiInformationLine } from '@remixicon/react';

export const InfoCallout = ({ text }: { text: string }) => (
  <div className="flex flex-row items-center justify-start rounded bg-sky-50 px-4 py-3">
    <RiInformationLine size={20} className="mr-2 text-sky-500" />
    <div className="text-base font-normal text-sky-700">{text}</div>
  </div>
);
