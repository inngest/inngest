import { init, type LDClient } from '@launchdarkly/node-server-sdk';

let launchDarklyClient: LDClient;

async function initialize() {
  const launchDarklySDKKey = process.env.LAUNCH_DARKLY_SDK_KEY;
  if (!launchDarklySDKKey) {
    throw new Error('LAUNCH_DARKLY_SDK_KEY environment variable is not set.');
  }
  const client = init(launchDarklySDKKey);
  await client.waitForInitialization();
  return client;
}

export async function getLaunchDarklyClient(): Promise<LDClient> {
  if (launchDarklyClient) return launchDarklyClient;
  return (launchDarklyClient = await initialize());
}
