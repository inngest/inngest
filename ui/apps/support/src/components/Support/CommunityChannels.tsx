import { Button } from "@inngest/components/Button";
import { Link } from "@inngest/components/Link";
import { RiGithubFill } from "@remixicon/react";

export function CommunityChannels() {
  return (
    <div className="mt-8 p-4 border rounded flex flex-col gap-4 max-w-xl">
      <h2 className="text-basis text-lg font-bold">Community</h2>
      <p>
        Chat with other developers and the Inngest team in our{" "}
        <Link
          target="_blank"
          href="https://www.inngest.com/discord"
          className="inline-flex"
          size="medium"
        >
          Discord community
        </Link>
        . Search for topics and questions in our{" "}
        <Link
          href="https://discord.com/channels/842170679536517141/1051516534029291581"
          className="inline-flex"
          target="_blank"
          size="medium"
        >
          #help-forum
        </Link>{" "}
        channel or submit your own question.
      </p>
      <Button
        kind="primary"
        href="https://www.inngest.com/discord"
        target="_blank"
        label="Join our Discord"
      />
      <h2 className="mt-4 text-basis text-lg font-bold">Open source</h2>
      <p>File an issue in our open source repos on Github:</p>
      <div>
        <p className="mb-2 text-sm font-medium">Inngest CLI + Dev Server</p>
        <Button
          appearance="outlined"
          kind="secondary"
          href="https://github.com/inngest/inngest/issues"
          label="inngest/inngest"
          icon={<RiGithubFill />}
          className="justify-start"
          iconSide="left"
        />
      </div>
      <div>
        <p className="mb-2 text-sm font-medium">SDKs</p>
        <div className="flex flex-row gap-2">
          <Button
            appearance="outlined"
            kind="secondary"
            href="https://github.com/inngest/inngest-js/issues"
            label="inngest/inngest-js"
            icon={<RiGithubFill />}
            iconSide="left"
          />
          <Button
            appearance="outlined"
            kind="secondary"
            href="https://github.com/inngest/inngest-py/issues"
            label="inngest/inngest-py"
            icon={<RiGithubFill />}
            iconSide="left"
          />
          <Button
            appearance="outlined"
            kind="secondary"
            href="https://github.com/inngest/inngestgo/issues"
            label="inngest/inngestgo"
            icon={<RiGithubFill />}
            iconSide="left"
          />
        </div>
      </div>
    </div>
  );
}
