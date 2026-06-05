import { useCallback, useEffect, useState } from 'react';

export type DynamicRunData = {
  runID: string;
  status: string;
  endedAt?: string;
};

type RunDataListener = {
  callback: (data: DynamicRunData | undefined) => void;
  runID?: string;
};

const dynamicRunDataEmitter = {
  listeners: new Set<RunDataListener>(),

  subscribe(callback: (step: DynamicRunData | undefined) => void, runID?: string) {
    const listener = { callback, runID };
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  },

  emit(data: DynamicRunData | undefined) {
    this.listeners.forEach((listener) => {
      if (!listener.runID || !data || listener.runID === data.runID) {
        listener.callback(data);
      }
    });
  },
};

//
// This is a way for the detail trace data (which we poll) to emit updates to statuses and
// run end times so we can immediately reflect those changes in the run list.
// This exists because we can't currently, easily poll the run list to get realtime updates there.
export const useDynamicRunData = ({ runID }: { runID?: string }) => {
  const [dynamicRunData, setDynamicRunData] = useState<DynamicRunData | undefined>(undefined);

  useEffect(() => {
    const cleanup = dynamicRunDataEmitter.subscribe(setDynamicRunData, runID);
    return () => {
      cleanup();
    };
  }, [runID]);

  const updateDynamicRunData = useCallback((data: DynamicRunData | undefined) => {
    dynamicRunDataEmitter.emit(data);
  }, []);

  return { dynamicRunData, updateDynamicRunData };
};
