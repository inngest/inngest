"use client";

import { useGetFunctionsQuery } from '@/store/generated';
import { BlankSlate } from '@/components/Blank';
import { useAppDispatch } from '@/store/hooks';
import { showDocs, showEventSendModal } from '@/store/global';
import { IconEvent, IconClock } from '@/icons';
import Skeleton from '@/components/Skeleton';
import Tag from '@/components/Tag';
import classNames from '@/utils/classnames';
import Button from '@/components/Button';

const cellStyles = 'pl-6 pr-2 py-3';

const HeaderCell = ({
  children,
  colSpan,
}: {
  children?: React.ReactNode;
  colSpan: number;
}) => {
  return (
    <th
      className={classNames(
        'w-fit whitespace-nowrap text-left text-xs font-semibold text-white',
        cellStyles
      )}
      colSpan={colSpan}
    >
      {children}
    </th>
  );
};

const TableSkeleton = () => {
  return (
    <>
      {[...Array(8)].map((_, index) => (
        <tr key={index}>
          <td className={classNames(cellStyles, 'max-h-5')} colSpan={3}>
            <Skeleton className="block h-5 w-32" />
          </td>
          <td className={classNames(cellStyles, 'max-h-5')} colSpan={2}>
            <Skeleton className="block h-5 w-32" />
          </td>
          <td className={classNames(cellStyles, 'max-h-5')} colSpan={3}>
            <Skeleton className="block h-5 w-48" />
          </td>
          <td className={classNames(cellStyles, 'max-h-5')} colSpan={1} />
        </tr>
      ))}
    </>
  );
};

export default function FunctionList() {
  const dispatch = useAppDispatch();

  const { data, isFetching } = useGetFunctionsQuery(undefined, {
    refetchOnMountOrArgChange: true,
  });
  const functions = data?.functions || [];

  return (
    <main className="flex min-h-0 flex-col overflow-y-auto">
      <table className="border-b border-slate-700/30 bg-slate-800/30 w-full table-fixed">
        <thead className="sticky top-0 shadow bg-slate-950">
          <tr>
            <HeaderCell colSpan={3}>Function Name</HeaderCell>
            <HeaderCell colSpan={2}>Triggers</HeaderCell>
            <HeaderCell colSpan={3}>App URL</HeaderCell>
            <HeaderCell colSpan={1} />
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800/30">
          {isFetching ? (
            <TableSkeleton />
          ) : functions?.length === 0 ? (
            <tr>
              <td className="p-10" colSpan={9}>
                <BlankSlate
                  title="Inngest has not detected any functions"
                  subtitle="Read our documentation to learn how to serve your functions"
                  imageUrl="/images/no-results.png"
                  button={{
                    text: 'Serving Functions',
                    onClick: () => dispatch(showDocs('/sdk/serve')),
                  }}
                />
              </td>
            </tr>
          ) : (
            <>
              {functions.map((func) => {
                const getFirstEventValue = () => {
                  const eventTrigger = func?.triggers?.find(
                    (trigger) => trigger.type === 'EVENT'
                  );
                  return eventTrigger ? eventTrigger.value : null;
                };
                const cleanUrl = new URL(func.url || '');
                cleanUrl.search = '';
                return (
                  <tr key={func.id}>
                    {/* Function Name */}
                    <td className="whitespace-nowrap" colSpan={3}>
                      <p
                        title={func.name}
                        className={classNames(
                          'pl-6 px-2 py-3 text-sm font-medium leading-7 text-ellipsis overflow-hidden',
                          cellStyles
                        )}
                      >
                        {func.name}
                      </p>
                    </td>
                    {/* Triggers */}
                    <td
                      className={classNames(cellStyles, 'whitespace-nowrap')}
                      colSpan={2}
                    >
                      {func.triggers?.map((trigger, index) => {
                        return (
                          <Tag key={index}>
                            <div className="flex items-center gap-2">
                              {trigger.type === 'EVENT' && (
                                <IconEvent className="h-2" />
                              )}
                              {trigger.type === 'CRON' && (
                                <IconClock className="h-3" />
                              )}
                              {trigger.value}
                            </div>
                          </Tag>
                        );
                      })}
                    </td>
                    {/* App URL */}
                    <td
                      className={classNames(cellStyles, 'whitespace-nowrap')}
                      colSpan={3}
                    >
                      {cleanUrl.toString()}
                    </td>
                    {/* Trigger Button */}
                    {getFirstEventValue() && (
                      <td
                        className={classNames(cellStyles, 'whitespace-nowrap')}
                        colSpan={1}
                      >
                        <Button
                          kind="secondary"
                          label="Trigger"
                          btnAction={() => {
                            dispatch(
                              showEventSendModal({
                                show: true,
                                data: JSON.stringify({
                                  name: getFirstEventValue(),
                                  data: {},
                                  user: {},
                                }),
                              })
                            );
                          }}
                        />
                      </td>
                    )}
                  </tr>
                );
              })}
            </>
          )}
        </tbody>
      </table>
    </main>
  );
};
