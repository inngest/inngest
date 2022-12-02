import { useState, useEffect, useMemo, useCallback } from "preact/hooks";
import TimeAgo, { Formatter, ReactTimeagoProps } from "react-timeago";

interface TimeProps extends ReactTimeagoProps {
  date: string | number;
}

const formatter: Formatter = (value, unit, suffix, ms, next) => {
  return next?.(value, unit, suffix, ms);
};

/**
 * TODO Add tooltip on hover with full date.
 */
export const Time = ({ date }: TimeProps) => {

  const tooltip = useMemo(() => new Date(date).toLocaleString(), [date]);

  // Refresh each second such that this component re-renders times appropriately.
  const [time, setTime] = useState(new Date().valueOf());
  useEffect(() => {
    const interval = setInterval(() => { setTime(new Date().valueOf()) }, 1000);
    return () => { clearInterval(interval); };
  }, []);

  return <TimeAgo date={date} formatter={formatter} title={tooltip} key={time} />;
};
