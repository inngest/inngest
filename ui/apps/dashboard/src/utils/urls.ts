export const WEBSITE_PRICING_URL = 'https://www.inngest.com/pricing';
export const WEBSITE_CONTACT_URL = 'https://www.inngest.com/contact';

export const DOCS_URLS = {
  SERVE: 'https://www.inngest.com/docs/sdk/serve',
};

export function getManageKey(pathname: string) {
  const regex = /\/manage\/(\w+)/;
  const match = pathname.match(regex);
  if (match && match[1]) {
    return match[1];
  } else {
    return null;
  }
}
