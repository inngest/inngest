import { useState } from 'react';
import Image from 'next/image';
import { AccordionList } from '@inngest/components/AccordionCard/AccordionList';
import { Button } from '@inngest/components/Button/Button';
import { Link } from '@inngest/components/Link';

import { ValidateModal } from '@/app/(organization-active)/(dashboard)/env/[environmentSlug]/apps/[externalID]/ValidateButton/ValidateModal';
import appActiveListDark from '@/images/app-active-list-dark.png';
import appActiveListLight from '@/images/app-active-list-light.jpg';

export default function AppFAQ() {
  const [showValidate, setShowValidate] = useState(false);

  return (
    <>
      <ValidateModal isOpen={showValidate} onClose={() => setShowValidate(false)} />

      <AccordionList className="" type="multiple" defaultValue={[]}>
        <AccordionList.Item value="no-app">
          <AccordionList.Trigger className="data-[state=open]:border-b-0">
            Unable to sync your first app?
          </AccordionList.Trigger>
          <AccordionList.Content>
            <div className="ml-5 flex items-center gap-6">
              <Image
                src={appActiveListLight}
                alt="screenshot of app list with synced app"
                className="hidden w-1/3 md:block dark:md:hidden"
              />
              <Image
                src={appActiveListDark}
                alt="screenshot of app list with synced app"
                className="hidden w-1/3 dark:md:block"
              />
              <div>
                <p className="text-muted mb-4 text-sm">
                  If your app is running but not appearing here, check its health status by clicking
                  on the button to diagnose any issues. If the issue persists, refer to our{' '}
                  <Link
                    href="https://www.inngest.com/docs/apps/cloud#troubleshooting?ref=apps-list-empty"
                    target="_blank"
                    className="inline"
                  >
                    documentation
                  </Link>
                  .
                </p>
                <Button
                  appearance="outlined"
                  onClick={() => setShowValidate(true)}
                  label="Check app health"
                  size="small"
                />
              </div>
            </div>
          </AccordionList.Content>
        </AccordionList.Item>
      </AccordionList>
    </>
  );
}
