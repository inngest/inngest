import colors from 'tailwindcss/colors';

type Indicator = 'none' | 'minor' | 'major' | 'critical';

export type StatusPageStatusResponse = {
  page: {
    id: string;
    name: string;
    url: string;
    updated_at: string;
  };
  status: {
    description: string;
    indicator: Indicator;
  };
};

export type Status = {
  url: string;
  description: string;
  indicator: Indicator;
  indicatorColor: string;
  updated_at: string;
};

// We use hex colors b/c tailwind only includes what is initially rendered
export const indicatorColor: { [K in Indicator]: string } = {
  none: colors.green['500'],
  minor: colors.yellow['300'],
  major: colors.orange['500'],
  critical: colors.red['600'],
};

export const STATUS_PAGE_URL = 'https://status.inngest.com';

export const mapStatus = (res: StatusPageStatusResponse) => ({
  ...res.status,
  indicatorColor: indicatorColor[res.status.indicator],
  updated_at: res.page.updated_at,
  url: res.page.url,
});

export const fetchStatus = async (): Promise<StatusPageStatusResponse> => {
  return await fetch('https://inngest.statuspage.io/api/v2/status.json').then((r) => r.json());
};

export const getStatus = async (): Promise<Status> => {
  return mapStatus(await fetchStatus());
};
