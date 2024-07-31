import { OptionalLink } from '@inngest/components/Link/OptionalLink';
import { RiArrowRightSLine } from '@remixicon/react';

import type { BreadCrumbType } from './Header';

export const BreadCrumb = ({ path }: { path: BreadCrumbType[] }) => {
  return path.map((part: BreadCrumbType, i: number) => {
    const last = i === path.length - 1;
    return (
      <div className="flex flex-row items-center justify-start" key={`${path}-key-${i}`}>
        <OptionalLink href={part.href}>
          <span className={`${last ? 'text-basis' : 'text-subtle'} mr-2 text-sm`}>{part.text}</span>
        </OptionalLink>

        {!last && <RiArrowRightSLine className="text-subtle mr-2 h-5 w-5" />}
      </div>
    );
  });
};
