import { EventStatus } from "../../store/generated";
import classNames from "../../utils/classnames";
import statusStyles from "../../utils/statusStyles";

interface TimelineFeedContent {
  date: Date;
  status: EventStatus;
  name: string;
  badge: number;
  active?: boolean;
  onClick?: () => void;
}

export default function TimelineFeedContent({
  date,
  status,
  name,
  badge,
  active = false,
  onClick,
}: TimelineFeedContent) {
  const eventStatusStyles = statusStyles(status);

  return (
    <a
      href=""
      className={classNames(
        active
          ? `outline outline-2 outline-indigo-400 outline-offset-3 bg-slate-900 border-slate-700/50`
          : `hover:bg-slate-800`,
        `pr-1.5 pl-2.5 pb-1.5 pt-2.5 bg-transparent border border-transparent text-left rounded group flex flex-col flex-1 min-w-0`
      )}
      onClick={
        onClick
          ? (e) => {
              e.preventDefault();
              onClick();
            }
          : undefined
      }
    >
      <span className="block text-3xs text-slate-300 pb-0.5">
        {date.toISOString()}
      </span>
      <div className="flex items-center">
        <h4 className="text-sm font-normal whitespace-nowrap overflow-hidden text-ellipsis grow pr-2 leading-none ">
          <span className={`${eventStatusStyles.text}`}>{name}</span>
        </h4>
        <span
          className={`rounded-md ${eventStatusStyles.fnBG} text-slate-100 text-3xs font-semibold leading-none flex items-center justify-center py-1.5 px-2`}
        >
          {badge}
        </span>
      </div>
    </a>
  );
}
