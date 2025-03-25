import { isStringArray } from '@inngest/components/utils/array';

import type { AppCheckResult } from '@/gql/graphql';

type Props = {
  data: AppCheckResult;
};

export function HTTPInfo({ data }: Props) {
  return (
    <>
      {!data.isReachable && (
        <div className="pl-3">No HTTP response since the app is unreachable</div>
      )}

      {data.respStatusCode && <div className="mb-4 pl-3">Status code: {data.respStatusCode}</div>}

      {data.respHeaders && Object.keys(data.respHeaders).length > 0 && (
        <table className="w-full">
          {Object.entries(data.respHeaders).map(([k, v]) => {
            if (!isStringArray(v)) {
              return null;
            }

            return (
              <tr className="border-subtle text-basis border-b text-sm" key={k}>
                <td className="px-3 py-1.5">{k}</td>
                <td className="py-1.5">{v.join(', ')}</td>
              </tr>
            );
          })}
        </table>
      )}
    </>
  );
}
