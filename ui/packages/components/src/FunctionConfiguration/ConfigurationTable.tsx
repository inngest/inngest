import type { InfoPopoverContent } from '@inngest/components/FunctionConfiguration/FunctionConfigurationInfoPopovers';
import { Info } from '@inngest/components/Info/Info';
import { Link } from '@inngest/components/Link';
import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { RiExternalLinkLine, RiInformationLine } from '@remixicon/react';

export type ConfigurationEntry = {
  label: string;
  value: React.ReactNode;
  type?: 'code';
};

type ConfigurationTableProps = {
  header: string;
  entries: ConfigurationEntry[];
  infoPopoverContent: InfoPopoverContent;
};

export default function ConfigurationTable({
  header,
  entries,
  infoPopoverContent,
}: ConfigurationTableProps) {
  if (entries.length == 0) {
    return null;
  }

  return (
    <div className="border-subtle overflow-hidden rounded border-[0.5px]">
      <table className="w-full table-fixed">
        <thead>
          <tr className="bg-disabled border-subtle h-8 border-b-[0.5px]">
            <td className="text-basis px-2 text-sm" colSpan={2}>
              <div className="flex items-center gap-2">
                {header}
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
              </div>
            </td>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr className="border-subtle h-8 border-b-[0.5px] last:border-b-0" key={entry.label}>
              <td className="text-muted px-2 text-sm">{entry.label}</td>
              <td className="text-basis px-2 text-right text-sm">
                {entry.type === 'code' ? (
                  <Tooltip>
                    <TooltipTrigger
                      asChild
                      className="text-muted block max-w-full truncate text-sm"
                    >
                      <code className="font-mono">{entry.value}</code>
                    </TooltipTrigger>
                    <TooltipContent className="max-w-md break-all p-3 text-sm">
                      <code>{entry.value}</code>
                    </TooltipContent>
                  </Tooltip>
                ) : (
                  entry.value
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
