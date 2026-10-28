import { toDate } from '@inngest/components/utils/date';
import { formatLocalTime, formatUTCTime } from '@inngest/components/utils/time';
import { toast } from 'sonner';

export const createTooltipTimestamp = (value: string) => {
  const header = document.createElement('div');
  header.className =
    'border-subtle bg-canvasBase sticky top-0 z-10 mb-2 flex items-start gap-3 border-b px-3 py-2';
  const date = toDate(value);

  if (!date) {
    header.textContent = value;
    return header;
  }

  const iso = date.toISOString();
  const times = document.createElement('div');
  times.className =
    'grid grid-cols-[auto_1fr] items-center gap-x-2 gap-y-1 text-xs';

  [
    { label: 'Local', value: formatLocalTime(date) },
    { label: 'UTC', value: formatUTCTime(date) },
  ].forEach(({ label, value }) => {
    const name = document.createElement('span');
    name.className = 'text-subtle';
    name.textContent = label;
    const time = document.createElement('time');
    time.dateTime = iso;
    time.textContent = value;
    times.append(name, time);
  });

  const button = document.createElement('button');
  button.type = 'button';
  button.className =
    'text-subtle hover:bg-canvasSubtle hover:text-basis focus-visible:ring-primary-moderate rounded p-1 focus-visible:outline-none focus-visible:ring-2';
  button.setAttribute('aria-label', 'Copy timestamp');
  button.title = `Copy ISO timestamp: ${iso}`;
  const copyIcon =
    '<svg aria-hidden="true" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="8" y="8" width="12" height="13" rx="1.5"/><path d="M16 8V3H3v13h5"/></svg>';
  button.innerHTML = copyIcon;
  let copying = false;
  button.addEventListener('click', async (event) => {
    event.stopPropagation();
    if (copying) return;
    copying = true;
    button.setAttribute('aria-disabled', 'true');
    button.innerHTML = copyIcon;
    button.setAttribute('aria-label', 'Copy timestamp');
    button.title = `Copy ISO timestamp: ${iso}`;
    try {
      await navigator.clipboard.writeText(iso);
      button.innerHTML =
        '<svg aria-hidden="true" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m5 12 4 4L19 6"/></svg>';
      button.setAttribute('aria-label', 'Timestamp copied');
      button.title = 'Copied to clipboard';
      toast.success('Copied to clipboard');
    } catch {
      toast.error('Could not copy timestamp');
    } finally {
      copying = false;
      button.removeAttribute('aria-disabled');
    }
  });

  header.append(times, button);
  return header;
};
