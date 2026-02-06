import { createColumnHelper } from '@tanstack/react-table';

import { LinkElement } from '../DetailsCard/Element';
import { useShared } from '../SharedContext/SharedContext';
import type { InngestStatus } from '../SharedContext/useInngestStatus';
import { getStatusBackgroundClass, getStatusTextClass } from '../Status/statusClasses';
import { Table } from '../Table';
import { OptionalTooltip } from '../Tooltip/OptionalTooltip';

type ErrorTable = {
  system: string;
  status: string;
  error: string;
};

type ErrorInfoProps = {
  error: string;
};

const InngestStatus = ({ inngestStatus }: { inngestStatus: InngestStatus | null }) =>
  inngestStatus && (
    <LinkElement href={inngestStatus.url} target="_blank">
      <span
        className={'mx-1 inline-flex h-2.5 w-2.5 rounded-full'}
        style={{ backgroundColor: inngestStatus.indicatorColor }}
      ></span>
      <div className="w-0 grow overflow-hidden text-ellipsis whitespace-nowrap">
        {inngestStatus.description}
      </div>
    </LinkElement>
  );

const SDKError = ({ error }: ErrorInfoProps) => (
  <div
    className={`flex min-w-0 items-center gap-2 rounded text-sm ${getStatusTextClass('FAILED')}`}
  >
    <div
      className={`mx-1 inline-flex h-2.5 w-2.5 shrink-0 rounded-full ${getStatusBackgroundClass(
        'FAILED'
      )}`}
    />
    <OptionalTooltip tooltip={error?.length > 55 ? error : ''} side="left">
      <div className="w-0 grow overflow-hidden text-ellipsis whitespace-nowrap">{error}</div>
    </OptionalTooltip>
  </div>
);

export const ErrorInfo = ({ error }: ErrorInfoProps) => {
  const { inngestStatus, cloud } = useShared();
  const columnHelper = createColumnHelper<ErrorTable>();

  const columns = [
    columnHelper.accessor('system', {
      cell: (info) => {
        const system = info.getValue();
        return <span className="text-muted text-sm leading-tight">{system}</span>;
      },
      header: 'System',
      enableSorting: false,
    }),
    columnHelper.accessor('status', {
      cell: ({ row }) => {
        const system = row.original.system;

        return system === 'Inngest' ? (
          <InngestStatus inngestStatus={inngestStatus} />
        ) : (
          <SDKError error={row.original.error} />
        );
      },
      header: 'Status',
      enableSorting: false,
    }),
  ];

  return (
    cloud && (
      <div className="my-2">
        <Table
          data={[
            { system: 'Inngest', status: '', error },
            { system: 'App', status: '', error },
          ]}
          columns={columns}
          cellClassName="[&:first-child]:w-24"
        />
      </div>
    )
  );
};
