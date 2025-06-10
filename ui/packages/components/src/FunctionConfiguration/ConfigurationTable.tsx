import { RiInformationLine } from '@remixicon/react';

export type ConfigurationEntry = {
  label: string;
  value: React.ReactNode;
};

type ConfigurationTableProps = {
  header: string;
  entries: ConfigurationEntry[];
};

export default function ConfigurationTable({ header, entries }: ConfigurationTableProps) {
  if (entries.length == 0) {
    return <></>;
  }

  return (
    <div className="overflow-hidden rounded border border-gray-300 ">
      <table className="w-full border-collapse">
        <thead>
          <tr className="h-8 border-b bg-gray-100 dark:bg-transparent">
            <td className="text-basis px-2 text-sm font-medium" colSpan={2}>
              <div className="flex items-center gap-2">
                {header}
                <RiInformationLine className="h-5 w-5" />
              </div>
            </td>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr className="h-8 border-b border-gray-200">
              <td className="text-muted px-2 text-sm">{entry.label}</td>
              <td className="text-basis px-2 text-right text-sm">{entry.value}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
