export const maxRenderedOutputSizeBytes = 1024 * 1024; // This prevents larger outputs from crashing the browser.
export enum FunctionRunExtraStatus {
  WaitingFor = 'WAITINGFOR',
  Sleeping = 'SLEEPING',
}
