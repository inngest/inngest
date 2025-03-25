import { usePathname } from 'next/navigation';

import { getManageKey } from '@/utils/urls';

export default function useManagePageTerminology() {
  const pathname = usePathname();
  const page = getManageKey(pathname);

  type ContentProps = {
    [key: string]: {
      name: string;
      type: string;
      param: string;
      titleType: string;
    };
  };

  const source: ContentProps = {
    keys: {
      name: 'Event Key',
      type: 'key',
      titleType: 'Key',
      param: 'keys',
    },
    webhooks: {
      name: 'Webhook',
      type: 'webhook',
      titleType: 'Webhook',
      param: 'webhooks',
    },
  };

  const currentContent = source[page as keyof ContentProps];

  return currentContent;
}
