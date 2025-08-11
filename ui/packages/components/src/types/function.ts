export type Function = {
  id: string;
  name: string;
  slug: string;
  triggers: {
    type: 'CRON' | 'EVENT';
    value: string;
  }[];
  version?: number | null;
  usage?: {
    totalVolume: number;
    dailyVolumeSlots: {
      startCount: number;
      failureCount: number;
    }[];
  };
  app?: {
    name: string;
    externalID: string;
  };
  isArchived?: boolean;
  isPaused?: boolean;
  failureRate?: number;
};

export type PageInfo = {
  currentPage: number;
  totalPages: number | null;
};
