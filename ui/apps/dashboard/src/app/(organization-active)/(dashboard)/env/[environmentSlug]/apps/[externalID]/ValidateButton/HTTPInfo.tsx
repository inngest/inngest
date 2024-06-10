import type { AppCheckResult } from '@/gql/graphql';
import isStringArray from '@/utils/isStringArray';

type Props = {
  data: AppCheckResult;
};

export function HTTPInfo({ data }: Props) {
  return (
    <>
      {!data.isReachable && <div>No HTTP response since the app is unreachable</div>}

      {data.respStatusCode && <div className="mb-4">Status code: {data.respStatusCode}</div>}

      {data.respHeaders && Object.keys(data.respHeaders).length > 0 && (
        <table className="w-full">
          {Object.entries(data.respHeaders).map(([k, v]) => {
            if (!isStringArray(v)) {
              return null;
            }

            return (
              <tr className="border-b border-slate-100" key={k}>
                <td className="py-1 pr-8">{k}</td>
                <td className="py-1">{v.join(', ')}</td>
              </tr>
            );
          })}
        </table>
      )}
    </>
  );
}
