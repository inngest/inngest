// This probably won't work for edge functions! When we start using feature
// flags in edge functions, we'll probably need to use
// @launchdarkly/vercel-server-sdk.
// import { init, type LDClient } from 'launchdarkly-node-server-sdk';
//
// let client: LDClient | undefined = undefined;
//
// export async function getLaunchDarklyClient(): Promise<LDClient> {
//   if (!client) {
//     const { LAUNCH_DARKLY_SDK_KEY } = process.env;
//     if (!LAUNCH_DARKLY_SDK_KEY) {
//       throw new Error('missing LAUNCH_DARKLY_SDK_KEY env var');
//     }
//
//     client = init(LAUNCH_DARKLY_SDK_KEY);
//     await client.waitForInitialization();
//   }
//
//   return client;
// }
