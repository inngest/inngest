import { Children } from 'react';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
import { RiExternalLinkLine, RiInformationLine } from '@remixicon/react';

import type { InfoPopoverContent } from './FunctionConfigurationTooltips';

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
      {/* TODO do we want font weight 450 specifically? */}
      <h3 className="text-basis mb-1 flex text-sm font-medium">
        {title}
        {infoPopoverContent && (
          <span className="flex items-center pl-1">
            <Info
              text={<span className="whitespace-pre-line">{infoPopoverContent.text}</span>}
              action={
                <Link
                  href={infoPopoverContent.url}
                  target="_blank"
                  iconAfter={<RiExternalLinkLine className="h-4 w-4" />}
                >
                  Learn more
                </Link>
              }
              IconComponent={RiInformationLine}
            />
          </span>
        )}
      </h3>

      {children}
    </div>
  );
}
