"use client";

import { useCallback, useMemo, useState, type UIEventHandler } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@inngest/components/Button";
import {
  IDCell,
  Table,
  TableBlankState,
  TextCell,
  TimeCell,
} from "@inngest/components/Table";
import {
  RiCloseCircleLine,
  RiDeleteBinLine,
  RiExternalLinkLine,
  RiRefreshLine,
} from "@remixicon/react";
import { createColumnHelper } from "@tanstack/react-table";

import { DeleteCancellationModal } from "./DeleteCancellationModal";
import { useCancellations } from "./useCancellations";

type Cancellation = {
  createdAt: string;
  envID: string;
  id: string;
  name: string | null;
  queuedAtMax: string;
  queuedAtMin: string | null;
};

type Props = {
  envSlug: string;
  fnSlug: string;
};

type PendingDelete = {
  id: string;
  envID: string;
};

export function CancellationTable({ envSlug, fnSlug }: Props) {
  const [pendingDelete, setPendingDelete] = useState<PendingDelete>();
  const columns = useColumns({ setPendingDelete });
  const router = useRouter();

  const {
    data: items,
    fetchNextPage,
    hasNextPage,
    isFetching,
    isInitiallyFetching,
  } = useCancellations({ envSlug, fnSlug });

  const onScroll: UIEventHandler<HTMLDivElement> = useCallback(
    (event) => {
      if (items.length > 0 && hasNextPage) {
        const { scrollHeight, scrollTop, clientHeight } =
          event.target as HTMLDivElement;

        // Check if scrolled to the bottom
        const reachedBottom = scrollHeight - scrollTop - clientHeight < 200;
        if (reachedBottom && !isFetching) {
          fetchNextPage();
        }
      }
    },
    [fetchNextPage, hasNextPage, items, isFetching],
  );

  return (
    <>
      <div className="bg-canvasBase text-basis no-scrollbar flex-1 overflow-hidden">
        <div className="h-full overflow-y-auto pb-2" onScroll={onScroll}>
          <Table
            blankState={
              <TableBlankState
                title="No cancellations found"
                icon={<RiCloseCircleLine />}
                actions={
                  <>
                    <Button
                      appearance="outlined"
                      label="Refresh"
                      onClick={() => router.refresh()}
                      icon={<RiRefreshLine />}
                      iconSide="left"
                    />
                    <Button
                      label="Go to docs"
                      href="https://www.inngest.com/docs/platform/manage/bulk-cancellation"
                      target="_blank"
                      icon={<RiExternalLinkLine />}
                      iconSide="left"
                    />
                  </>
                }
              />
            }
            columns={columns}
            data={items}
            isLoading={isInitiallyFetching}
          />
        </div>
      </div>
      <DeleteCancellationModal
        onClose={() => setPendingDelete(undefined)}
        pendingDelete={pendingDelete}
      />
    </>
  );
}

const columnHelper = createColumnHelper<Cancellation>();

function useColumns({
  setPendingDelete,
}: {
  setPendingDelete: (obj: PendingDelete) => void;
}) {
  return useMemo(() => {
    return [
      columnHelper.accessor("name", {
        header: "Name",
        cell: (props) => {
          return <TextCell>{props.getValue()}</TextCell>;
        },
        enableSorting: false,
      }),
      columnHelper.accessor("createdAt", {
        header: "Created at",
        cell: (props) => {
          return <TimeCell date={props.getValue()} />;
        },
        enableSorting: false,
      }),
      columnHelper.accessor("id", {
        header: "ID",
        cell: (props) => {
          return <IDCell>{props.getValue()}</IDCell>;
        },
        enableSorting: false,
      }),
      columnHelper.accessor("queuedAtMin", {
        header: "Minimum queued at (filter)",
        cell: (props) => {
          const value = props.getValue();
          if (!value) {
            return <span>-</span>;
          }

          return <TimeCell date={value} />;
        },
        enableSorting: false,
      }),
      columnHelper.accessor("queuedAtMax", {
        header: "Maximum queued at (filter)",
        cell: (props) => {
          return <TimeCell date={props.getValue()} />;
        },
        enableSorting: false,
      }),
      columnHelper.display({
        id: "actions",
        header: undefined, // Needed to enable the iconOnly styles in the table
        cell: (props) => {
          const data = props.row.original;

          return (
            <Button
              appearance="ghost"
              icon={<RiDeleteBinLine className="size-5" />}
              kind="danger"
              onClick={() => setPendingDelete(data)}
            />
          );
        },
        enableSorting: false,
      }),
    ];
  }, [setPendingDelete]);
}
