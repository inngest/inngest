export function isMenuItemActive(currentHref: string, targetHref: string, exact = false): boolean {
  if (!targetHref) {
    return false;
  }

  if (!exact) {
    return currentHref.startsWith(targetHref);
  }

  return normalizeHref(currentHref) === normalizeHref(targetHref);
}

function normalizeHref(href: string): string {
  const path = href.split(/[?#]/)[0] || '';
  if (path.length > 1) {
    return path.replace(/\/+$/, '');
  }

  return path;
}
