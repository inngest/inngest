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
  return <TimeAgo date={date} formatter={formatter} />;
};
