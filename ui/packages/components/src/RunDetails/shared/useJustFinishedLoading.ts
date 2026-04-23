import { useEffect, useRef, useState } from 'react';

export const useJustFinishedLoading = (loading: boolean, durationMs = 2000): boolean => {
  const [justFinished, setJustFinished] = useState(false);
  const prevLoading = useRef(loading);

  useEffect(() => {
    if (prevLoading.current && !loading) {
      setJustFinished(true);
      const t = setTimeout(() => setJustFinished(false), durationMs);
      prevLoading.current = loading;
      return () => clearTimeout(t);
    }
    prevLoading.current = loading;
  }, [loading, durationMs]);

  return justFinished;
};
