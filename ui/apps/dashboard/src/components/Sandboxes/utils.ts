import { SandboxStatus } from '@/gql/graphql';

export function sandboxStatus(status: SandboxStatus): {
  colorStatus: string;
  label: string;
} {
  switch (status) {
    case SandboxStatus.Pending:
      return { colorStatus: 'QUEUED', label: 'Starting' };
    case SandboxStatus.Running:
      return { colorStatus: 'RUNNING', label: 'Running' };
    case SandboxStatus.Stopping:
      return { colorStatus: 'RUNNING', label: 'Stopping' };
    case SandboxStatus.Stopped:
      return { colorStatus: 'COMPLETED', label: 'Stopped' };
    case SandboxStatus.Failed:
      return { colorStatus: 'FAILED', label: 'Failed' };
    case SandboxStatus.LaunchUnknown:
      return { colorStatus: 'UNKNOWN', label: 'Launch unknown' };
  }
}

export function formatMemory(memoryMB: number): string {
  if (memoryMB >= 1024 && memoryMB % 1024 === 0) {
    return `${memoryMB / 1024} GB`;
  }
  return `${memoryMB.toLocaleString()} MB`;
}

export function formatDuration(milliseconds: number): string {
  if (milliseconds < 1_000) return `${Math.round(milliseconds)}ms`;
  if (milliseconds < 60_000) return `${(milliseconds / 1_000).toFixed(1)}s`;
  return `${Math.round(milliseconds / 60_000)}m`;
}

export function formatTimeout(seconds: number): string {
  if (seconds === 0) return 'No timeout';
  return formatDuration(seconds * 1_000);
}

export function compactSandboxID(id: string): string {
  return id.slice(0, 8);
}
