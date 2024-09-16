import { NewButton } from '../Button';
import { NewLink } from '../Link';
import { STEPS_ORDER, type ConnectPostgresIntegrationContent } from './types';

export default function ConnectPage({
  content,
  onStartInstallation,
}: {
  content: ConnectPostgresIntegrationContent;
  onStartInstallation: () => void;
}) {
  const { title, description, logo, url, step } = content;
  return (
    <div className="mx-auto mt-16 flex w-[800px] flex-col gap-9">
      <div>
        <div className="text-basis mb-7 flex flex-row items-center justify-start text-2xl font-medium">
          <div className="bg-contrast mr-4 flex h-12 w-12 items-center justify-center rounded">
            {logo}
          </div>
          <h2 className="text-basis text-xl">{title}</h2>
        </div>
        <p className="text-subtle text-sm">
          {description}
          {url && (
            <NewLink size="small" className="ml-1 inline-block" href={url}>
              Read documentation
            </NewLink>
          )}
        </p>
      </div>
      <p className="font-lg text-basis">Installation overview</p>
      <div>
        {STEPS_ORDER.map((stepKey, index) => {
          const stepContent = step[stepKey];
          const isLastStep = index === STEPS_ORDER.length - 1;
          return (
            <div key={stepKey} className={`border-subtle ml-3 ${!isLastStep ? 'border-l' : ''} `}>
              <div
                className="before:border-subtle before:text-light before:bg-canvasBase relative ml-[32px] pb-7 before:absolute before:left-[-46px] before:flex before:h-[28px] before:w-[28px] before:items-center before:justify-center before:rounded-full before:border before:text-[13px] before:content-[attr(data-step-number)]"
                data-step-number={index + 1}
              >
                <div className="text-basis text-base">{stepContent.title}</div>
                <div className="text-subtle text-sm">{stepContent.description}</div>
              </div>
            </div>
          );
        })}
      </div>
      <div>
        <NewButton label="Start installation" onClick={onStartInstallation} />
      </div>
    </div>
  );
}
