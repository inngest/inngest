import { NewButton } from '@inngest/components/Button';
import { InfoCallout } from '@inngest/components/Callout';
import { Link } from '@inngest/components/Link';
import { IconVercel } from '@inngest/components/icons/platforms/Vercel';

export default function VercelConnect() {
  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col">
      <div className="mb-7 flex flex-row items-center justify-start text-2xl font-medium">
        <div className="mr-4 flex h-12 w-12 items-center justify-center rounded bg-black">
          <IconVercel className="text-white" size={20} />
        </div>
        Vercel
      </div>
      <div className="mb-7 w-full text-base font-normal text-slate-500">
        This integration enables you to host your Inngest functions on the Vercel platform and
        automatically sync them every time you deploy code.{' '}
        <Link showIcon={false} href="https://www.inngest.com/docs/deploy/vercel">
          Read documentation
        </Link>
      </div>
      <div className="mb-7">
        <InfoCallout text="Please note that each Vercel account can only be linked to one Inngest account at a time." />
      </div>
      <div className="mb-7 text-lg font-normal text-slate-800">Installation overview</div>
      <div className="text-lg font-normal text-slate-800">
        <div className="ml-3 border-l border-slate-300">
          <div className="relative ml-[32px] pb-7 before:absolute before:left-[-46px] before:h-[28px] before:w-[28px] before:rounded-full before:border before:border-slate-300 before:bg-white before:text-center before:align-middle before:text-[13px] before:text-slate-700 before:content-['1']">
            <div className="leading-6 text-slate-950">Install Inngest Integration on Vercel.</div>
            <div className="leading-6 text-slate-500">
              Click the &rdquo;Add Integration&rdquo; button.
            </div>
          </div>
        </div>

        <div className="ml-3 border-l border-slate-300">
          <div className="relative ml-[32px] pb-7 before:absolute before:left-[-46px] before:h-[28px] before:w-[28px] before:rounded-full before:border before:border-slate-300 before:bg-white before:text-center before:align-middle before:text-[13px] before:text-slate-700 before:content-['2']">
            <div className="leading-8 text-slate-950">
              Select the Vercel projects you wish to enable
            </div>
            <div className="leading-6 text-slate-500">
              You can configure one or more serve endpoints.
            </div>
          </div>
        </div>
        <div className="ml-3">
          <div className="relative ml-[32px] pb-7 before:absolute before:left-[-46px] before:h-[28px] before:w-[28px] before:rounded-full before:border before:border-slate-300 before:bg-white before:text-center before:align-middle before:text-[13px] before:text-slate-700 before:content-['3']">
            <div className="leading-6 text-slate-950">Setup Successful</div>
            <div className="leading-6 text-slate-500">
              The integration auto-configures necessary environment variables and syncs your app
              with Inngest whenever you deploy code to Vercel.
            </div>
          </div>
        </div>
      </div>
      <div>
        <NewButton
          appearance="solid"
          href="https://vercel.com/integrations/inngest"
          label="Connect Vercel to Inngest"
        />
      </div>
    </div>
  );
}
