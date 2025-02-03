import { AppsIcon } from '@inngest/components/icons/sections/Apps';

type EmptyAppsCardProps = {
  title: string;
  description: string | React.ReactNode;
  actions: React.ReactNode;
};

export default function EmptyAppsCard({ title, description, actions }: EmptyAppsCardProps) {
  return (
    <div className="border-muted bg-canvasBase text-basis mb-6 flex flex-col items-center gap-5 rounded-md border border-dashed px-6 py-9">
      <div className="bg-canvasSubtle text-light rounded-md p-3 ">
        <AppsIcon className="h-7 w-7" />
      </div>
      <div className="text-center">
        <p className="mb-2 text-xl">{title}</p>
        <p className="text-subtle max-w-xl text-sm">{description}</p>
      </div>
      <div className="flex items-center gap-3">{actions}</div>
    </div>
  );
}
