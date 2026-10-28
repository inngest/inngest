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
    case SandboxStatus.Pausing:
      return { colorStatus: 'RUNNING', label: 'Pausing' };
    case SandboxStatus.Paused:
      return { colorStatus: 'PAUSED', label: 'Paused' };
    case SandboxStatus.Resuming:
      return { colorStatus: 'QUEUED', label: 'Resuming' };
    case SandboxStatus.Stopping:
      return { colorStatus: 'RUNNING', label: 'Stopping' };
    case SandboxStatus.Stopped:
      return { colorStatus: 'CANCELLED', label: 'Stopped' };
    case SandboxStatus.Failed:
      return { colorStatus: 'FAILED', label: 'Failed' };
    case SandboxStatus.LaunchUnknown:
      return { colorStatus: 'UNKNOWN', label: 'Launch unknown' };
    default:
      // Statuses added to the API before the dashboard knows about them.
      return { colorStatus: 'UNKNOWN', label: 'Unknown' };
  }
}

export function isActiveSandboxStatus(status: SandboxStatus): boolean {
  return (
    status === SandboxStatus.Pending ||
    status === SandboxStatus.Running ||
    status === SandboxStatus.Stopping
  );
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

  const minutes = Math.round(milliseconds / 60_000);
  if (minutes < 60) return `${minutes}m`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return joinUnits(hours, 'h', minutes % 60, 'm');
  return joinUnits(Math.floor(hours / 24), 'd', hours % 24, 'h');
}

function joinUnits(
  major: number,
  majorUnit: string,
  minor: number,
  minorUnit: string,
): string {
  return minor
    ? `${major}${majorUnit} ${minor}${minorUnit}`
    : `${major}${majorUnit}`;
}

export function formatTimeout(seconds: number): string {
  if (seconds === 0) return 'No timeout';
  return formatDuration(seconds * 1_000);
}

export function compactSandboxID(id: string): string {
  return id.slice(0, 8);
}
