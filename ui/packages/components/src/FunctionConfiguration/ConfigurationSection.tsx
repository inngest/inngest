import { Children } from 'react';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
import { RiExternalLinkLine, RiInformationLine } from '@remixicon/react';

import type { InfoPopoverContent } from './FunctionConfigurationInfoPopovers';

type ConfigurationSectionProps = {
  title?: string;
  children: React.ReactNode;
  infoPopoverContent?: InfoPopoverContent;
};

export default function ConfigurationSection({
  title,
  children,
  infoPopoverContent,
}: ConfigurationSectionProps) {
  if (Children.toArray(children).length == 0) {
    return null;
  }

  return (
    <div>
      <h3 className="text-basis mb-1 flex text-sm">
        {title}
        {infoPopoverContent && (
          <span className="text-muted flex items-center pl-1">
            <Info
              text={<span className="whitespace-pre-line">{infoPopoverContent.text}</span>}
              widthClassName="max-w-60"
              action={
                <Link
                  href={infoPopoverContent.url}
                  target="_blank"
                  iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
                >
                  Learn more
                </Link>
              }
              iconElement={<RiInformationLine className="text-muted h-[18px] w-[18px]" />}
            />
          </span>
        )}
      </h3>
      <div>{children}</div>
    </div>
  );
}
