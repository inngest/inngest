import { useMemo } from "preact/hooks";
import TimeAgo, { Formatter, ReactTimeagoProps } from "react-timeago";

interface TimeProps extends ReactTimeagoProps {
  date: string | number;
}

const formatter: Formatter = (value, unit, suffix, ms, next) => {
  if (unit === "second") {
    return "less then a minute ago";
  }

  return next?.(value, unit, suffix, ms);
};

/**
 * TODO Add tooltip on hover with full date.
 */
export const Time = ({ date }: TimeProps) => {
  const tooltip = useMemo(() => new Date(date).toLocaleString(), [date]);

  return <TimeAgo date={date} formatter={formatter} title={tooltip} />;
};
