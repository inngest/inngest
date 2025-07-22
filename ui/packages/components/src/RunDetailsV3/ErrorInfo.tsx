import { createColumnHelper } from '@tanstack/react-table';

import { useShared } from '../SharedContext/SharedContext';
import type { InngestStatus } from '../SharedContext/useInngestStatus';
import { getStatusBackgroundClass, getStatusTextClass } from '../Status/statusClasses';
import NewTable from '../Table/NewTable';

type ErrorTable = {
  system: string;
  status: string;
};

type ErrorInfoProps = {
  error: string;
};

const InngestStatus = ({ inngestStatus }: { inngestStatus: InngestStatus | null }) =>
  inngestStatus && (
    <a
      href={inngestStatus.url}
      target="_blank"
      className="hover:text-link bg-canvasBase hover:bg-canvasMuted text-basis flex items-center gap-2 rounded text-sm"
    >
      <span
        className={'mx-1 inline-flex h-2.5 w-2.5 rounded-full'}
        style={{ backgroundColor: inngestStatus.indicatorColor }}
      ></span>
      {inngestStatus.description}
    </a>
  );

const VercelStatus = ({ inngestStatus }: { inngestStatus: InngestStatus | null }) =>
  inngestStatus && (
    <a
      href={inngestStatus.url}
      target="_blank"
      className="hover:text-link bg-canvasBase hover:bg-canvasMuted text-basis flex items-center gap-2 rounded text-sm"
    >
      <div
        className={'mx-1 inline-flex h-2.5 w-2.5 rounded-full'}
        style={{ backgroundColor: inngestStatus.indicatorColor }}
      />
      {inngestStatus.description}
    </a>
  );

const SDKError = ({ error }: ErrorInfoProps) => (
  <div className={`flex items-center gap-2 rounded text-sm ${getStatusTextClass('FAILED')}`}>
    <div
      className={`mx-1 inline-flex h-2.5 w-2.5 shrink-0 rounded-full ${getStatusBackgroundClass(
        'FAILED'
      )}`}
    />
    <div className="min-w-0 overflow-x-auto whitespace-nowrap">{error}</div>
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
      size: 25,
      enableSorting: false,
    }),
    columnHelper.accessor('status', {
      cell: (row) => {
        const system = row.row.original.system;

        return system === 'Inngest' ? (
          <InngestStatus inngestStatus={inngestStatus} />
        ) : system === 'Vercel' ? null : ( // <VercelStatus inngestStatus={inngestStatus} />
          <SDKError error={error} />
        );
      },
      header: 'Status',
      enableSorting: false,
    }),
  ];

  return (
    cloud && (
      <div>
        <NewTable
          data={[
            { system: 'Inngest', status: '' },
            { system: 'SDK', status: '' },
          ]}
          columns={columns}
        />
      </div>
    )
  );
};
