import type { InfoPopoverContent } from '@inngest/components/FunctionConfiguration/FunctionConfigurationTooltips';
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
    <div className="border-subtle overflow-hidden rounded border">
      <table className="w-full table-fixed border-collapse">
        <thead>
          <tr className="bg-disabled h-8 border-b">
            <td className="text-basis px-2 text-sm font-medium" colSpan={2}>
              <div className="flex items-center gap-2">
                {header}
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
              </div>
            </td>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr className="h-8 border-b" key={entry.label}>
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
                    <TooltipContent className="text-muted bg-canvasBase border-subtle border p-3 text-sm">
                      <div>
                        <h2 className="text-basis gap-1 text-xs">Expression</h2>
                        <div className="bg-codeEditor border-subtle text-muted flex items-start gap-2 self-stretch border px-3 text-xs leading-5">
                          {entry.value}
                        </div>
                      </div>
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
