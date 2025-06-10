import { Tooltip, TooltipContent, TooltipTrigger } from '@inngest/components/Tooltip';
import { RiInformationLine } from '@remixicon/react';

export type ConfigurationEntry = {
  label: string;
  value: React.ReactNode;
  type?: 'code';
};

type ConfigurationTableProps = {
  header: string;
  entries: ConfigurationEntry[];
};

export default function ConfigurationTable({ header, entries }: ConfigurationTableProps) {
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
                <RiInformationLine className="bg-canvasSubtle h-5 w-5" />
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
