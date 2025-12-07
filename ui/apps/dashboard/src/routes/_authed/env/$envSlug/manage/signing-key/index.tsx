import { createFileRoute } from "@tanstack/react-router";
import { useEnvironment } from "@/components/Environments/environment-context";
import { CreateSigningKeyButton } from "@/components/SigningKey/CreateSigningKeyButton";
import { RotateSigningKeyButton } from "@/components/SigningKey/RotateSigningKeyButton";
import { SigningKey } from "@/components/SigningKey/SigningKey";
import { useSigningKeys } from "@/components/SigningKey/useSigningKeys";
import { EnvironmentType } from "@/gql/graphql";
import { Alert } from "@inngest/components/Alert/NewAlert";
import { Card } from "@inngest/components/Card";
import { InlineCode } from "@inngest/components/Code";
import LoadingIcon from "@/components/Icons/LoadingIcon";

export const Route = createFileRoute(
  "/_authed/env/$envSlug/manage/signing-key/",
)({
  component: SigningKeyComponent,
});

function SigningKeyComponent() {
  const env = useEnvironment();

  const { data, error, isLoading } = useSigningKeys({ envID: env.id });
  if (error) {
    throw error;
  }
  if (isLoading && !data) {
    return (
      <div className="flex h-full w-full items-center justify-center">
        <LoadingIcon />
      </div>
    );
  }

  const activeKeys = data.filter((key) => key.isActive);
  if (!activeKeys[0]) {
    // Unreachable
    throw new Error("No active key found");
  }
  if (activeKeys.length > 1) {
    // Unreachable
    throw new Error("More than one active key found");
  }
  const activeKey = activeKeys[0];
  const inactiveKeys = data.filter((key) => !key.isActive);

  return (
    <div className="flex min-h-0 flex-1">
      <div className="text-basis h-full min-w-0 flex-1 overflow-y-auto">
        <div>
          {env.type === EnvironmentType.BranchParent && (
            <Alert
              className="flex items-center justify-center rounded-none text-sm"
              severity="info"
            >
              Signing keys are shared for all branch environments
            </Alert>
          )}
          <div className="my-8 flex items-center justify-center">
            <div className="divide-subtle w-full max-w-[800px] divide-y">
              <div className="mb-8">
                <h1 className="mb-2 text-2xl">Signing keys</h1>

                <p className="text-muted mb-8 text-sm">
                  Signing keys are secrets used for secure communication between
                  Inngest and your apps.
                  <a
                    target="_blank"
                    href="https://www.inngest.com/docs/security#signing-keys-and-sdk-security"
                  >
                    Learn More
                  </a>
                </p>

                <SigningKey signingKey={activeKey} />

                {inactiveKeys.map((signingKey) => {
                  return (
                    <SigningKey key={signingKey.id} signingKey={signingKey} />
                  );
                })}

                <CreateSigningKeyButton
                  disabled={data.length > 1}
                  envID={env.id}
                />
              </div>

              <div>
                <h2 className="mb-2 mt-4 text-xl">Rotation</h2>

                <div className="text-subtle mb-8 text-sm">
                  Create a new signing key and update environment variables in
                  your app: set <InlineCode>INNGEST_SIGNING_KEY</InlineCode> to
                  the value of the <span className="font-bold">new key</span>{" "}
                  and <InlineCode>INNGEST_SIGNING_KEY_FALLBACK</InlineCode> to
                  the value of the{" "}
                  <span className="font-bold">current key</span>. Deploy your
                  apps and then click the{" "}
                  <span className="font-bold">Rotate key</span> button.
                </div>

                <Card>
                  <Card.Content className="p-4">
                    <div className="mb-4 flex items-center">
                      <div className="grow">
                        <p className="mb-2 font-medium">Rotate key</p>

                        <p className="text-subtle text-sm">
                          This action replaces the{" "}
                          <span className="font-bold">current key</span> with
                          the <span className="font-bold">new key</span>,
                          permanently deleting the current key.
                        </p>
                      </div>

                      <RotateSigningKeyButton
                        disabled={inactiveKeys.length === 0}
                        envID={env.id}
                      />
                    </div>

                    <Alert severity="warning" className="text-sm">
                      <p>
                        Rotation may cause downtime if your SDK does not meet
                        the minimum version.
                        <Alert.Link
                          severity="warning"
                          target="_blank"
                          href="https://www.inngest.com/docs/platform/signing-keys#rotation"
                        >
                          Learn More
                        </Alert.Link>
                      </p>
                    </Alert>
                  </Card.Content>
                </Card>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
