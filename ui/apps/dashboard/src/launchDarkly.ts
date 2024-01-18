import { init, type LDClient } from '@launchdarkly/node-server-sdk';

let launchDarklyClient: LDClient | undefined;

function initialize() {
  const launchDarklySDKKey = process.env.LAUNCH_DARKLY_SDK_KEY;
  if (!launchDarklySDKKey) {
    throw new Error('LAUNCH_DARKLY_SDK_KEY environment variable is not set.');
  }
  launchDarklyClient = init(launchDarklySDKKey, { stream: false });
  return launchDarklyClient;
}

export async function getLaunchDarklyClient(): Promise<LDClient> {
  if (!launchDarklyClient) {
    return initialize();
  }

  await launchDarklyClient.waitForInitialization();
  return launchDarklyClient;
}
