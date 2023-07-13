import { useGetFunctionsQuery, type Function } from "../store/generated";
import { BlankSlate } from "../components/Blank";
import { useAppDispatch } from "../store/hooks";
import { showDocs } from "../store/global";
import { IconEvent, IconClock } from '@/icons';
import Skeleton from '@/components/Skeleton';
import Tag from '@/components/Tag';
import classNames from '@/utils/classnames';

const cellStyles = 'pl-6 pr-2 py-3';

const HeaderCell = ({ children }: { children: React.ReactNode }) => {
  return (
    <th
      className={classNames(
        'w-fit whitespace-nowrap text-left text-xs font-semibold text-white',
        cellStyles
      )}
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
          <td className={classNames(cellStyles, 'max-h-5' )}>
            <Skeleton className="block h-5 w-32" />
          </td>
          <td className={classNames(cellStyles, 'max-h-5' )}>
            <Skeleton className="block h-5 w-32" />
          </td>
          <td className={classNames(cellStyles, 'max-h-5' )}>
            <Skeleton className="block h-5 w-48" />
          </td>
        </tr>
      ))}
    </>
  );
};


export const FunctionList = () => {
  const dispatch = useAppDispatch();

  const { data, isFetching } = useGetFunctionsQuery(undefined, {refetchOnMountOrArgChange: true});
  const functions = data?.functions || [];

  return (
    <main className="flex min-h-0 flex-col overflow-y-auto">
      <table className="border-b border-slate-700/30 bg-slate-800/30 table-fixed w-full">
        <thead className="sticky top-0 shadow bg-slate-950">
          <tr>
            <HeaderCell>Function Name</HeaderCell>
            <HeaderCell>Triggers</HeaderCell>
            <HeaderCell>App URL</HeaderCell>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800/30">
          {isFetching ? (
            <TableSkeleton />
          ) : functions?.length === 0 ? (
            <tr>
              <td className="p-10" colSpan={3}>
                <BlankSlate
                  title="Inngest has not detected any functions"
                  subtitle="Read our documentation to learn how to serve your functions"
                  imageUrl="/images/no-results.png"
                  button={{
                    text: 'Serving Functions',
                    onClick: () => dispatch(showDocs("/sdk/serve")),
                  }}
                />
              </td>
            </tr>
          ) : (
            <>
              {functions.map((func) => {
                const cleanUrl = new URL(func.url || '');
                cleanUrl.search = '';
                return (
                  <tr key={func.id}>
                    {/* Function Name */}
                    <td className="whitespace-nowrap">
                      <p
                        className={classNames(
                          'pl-6 px-2 py-3 text-sm font-medium leading-7',
                          cellStyles
                        )}
                      >
                        {func.name}
                      </p>
                    </td>
                    {/* Triggers */}
                    <td className={classNames(cellStyles, 'whitespace-nowrap')}>
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
                    <td className={classNames(cellStyles, 'whitespace-nowrap')}>{cleanUrl.toString()}</td>
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
