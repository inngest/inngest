import { useRouter } from '@tanstack/react-router';

export function publicHref(href: string | undefined, basepath = '/'): string | undefined {
  const base = basepath.replace(/\/$/, '');
  if (
    !href ||
    !base ||
    !href.startsWith('/') ||
    href.startsWith('//') ||
    href === base ||
    href.startsWith(`${base}/`) ||
    href.startsWith(`${base}?`) ||
    href.startsWith(`${base}#`)
  )
    return href;
  return `${base}${href}`;
}

// plain anchors do not add the router's base path.  this keeps links opened
// in new tabs under /dev when the UI is hosted on the website.
export function usePublicHref(href?: string): string | undefined {
  const router = useRouter({ warn: false });
  return publicHref(href, router?.basepath);
}
