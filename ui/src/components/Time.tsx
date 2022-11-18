import TimeAgo, { ReactTimeagoProps } from "react-timeago";

interface TimeProps extends ReactTimeagoProps {
  date: string | number;
}

/**
 * TODO Add tooltip on hover with full date.
 */
export const Time = ({ date }: TimeProps) => {
  return <TimeAgo date={date} />;
};
