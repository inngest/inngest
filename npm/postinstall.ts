#!/usr/bin/env node
import AdmZip from "adm-zip";
import Debug from "debug";
import fetch, { Response } from "node-fetch";
import path from "path";
import tar from "tar";
import { URL } from "url";
import packageJson from "./package.json";

const rootDebug = Debug("inngest:cli");

/**
 * A map of Node's `process.arch` to Golang's `$GOARCH` equivalent.
 */
const archMap: Partial<Record<typeof process.arch, string>> = {
  arm64: "arm64",
  x64: "amd64",
};

/**
 * Known extensions that we package our binaries with.
 *
 * In an enum so that we can be type-safe with Record<> and ensure we're always
 * handling any extensions we add here.
 */
enum KnownExtension {
  Tar = ".tar.gz",
  Zip = ".zip",
}

/**
 * A map of Node's `process.platform` to Golang's `$GOOS` equivalent and the
 * extension of the packaged binary we provide.
 */
const platformMap: Partial<
  Record<
    typeof process.platform,
    { platform: string; extension: KnownExtension }
  >
> = {
  linux: { platform: "linux", extension: KnownExtension.Tar },
  darwin: { platform: "darwin", extension: KnownExtension.Tar },
  win32: { platform: "windows", extension: KnownExtension.Zip },
};

/**
 * Fetches the supposed URL of the binary on GitHub for this version.
 */
async function getBinaryUrl(): Promise<URL> {
  const debug = rootDebug.extend("getBinaryUrl");

  const { arch, platform } = getArchPlatform();

  debug({ arch, platform });

  let version = packageJson.version.trim();
  debug("package.json version:", version);

  const targetUrl = new URL(
    `https://cli.inngest.com/artifact/v${version}/inngest_${version}_${platform.platform}_${arch}${platform.extension}`
  );

  debug("targetUrl:", targetUrl.href);

  return targetUrl;
}

/**
 * Return the platform and architecture included in the supposed binary name
 * from `$GOOS` and `$GOARCH`.
 *
 * @throws when the arch or platform are not supported by this script.
 */
function getArchPlatform() {
  const debug = rootDebug.extend("getArchPlatform");

  const arch = archMap[process.arch];
  const platform = platformMap[process.platform];

  debug({
    arch,
    platform,
    "process.arch": process.arch,
    "process.platform": process.platform,
  });

  if (!arch) {
    throw new Error(`Unsupported architecture: ${process.arch}`);
  }

  if (!platform) {
    throw new Error(`Unsupported platform: ${process.platform}`);
  }

  return { arch, platform };
}

/**
 * Attempts to fetch a binary given a `url` and returns the response so long as
 * it is a successful `200 OK`.
 *
 * @throws when the response is not `200 OK`.
 */
function downloadBinary(
  /**
   * The URL to download the binary from.
   */
  url: URL
): Promise<Response> {
  const debug = rootDebug.extend("downloadBinary");

  /**
   * Using `new Promise()` here so that we can use listeners when dealing with
   * the request in order to ensure it's valid without using the body.
   */
  return new Promise(async (resolve, reject) => {
    debug("downloading binary from:", url.href);

    fetch(url.href, {
      redirect: "follow",
    })
      .then((res) => {
        if (res.status !== 200) {
          throw new Error(
            `Error downloading binary; invalid response status code: ${res.status}`
          );
        }

        resolve(res);
      })
      .catch(reject);
  });
}

/**
 * Takes a `req` and pipes it through unpacking and saving to the local `bin`
 * dir within the package contents.
 *
 * @throws if creating the target location or piping fails
 */
function pipeBinaryToInstallLocation(
  /**
   * The unused request to download a binary.
   */
  res: Response,

  /**
   * The original URL used to access the binary.
   *
   * We need this to determine the extension of the binary to decide how to
   * safely unpack it on this platform/arch. We can't use `req.path` for this
   * because that can be mutated if we're redirected to a different URL to
   * perform the download.
   */
  originalUrl: URL
): Promise<void> {
  const debug = rootDebug.extend("pipeBinaryToInstallLocation");

  return new Promise<void>((resolve, reject) => {
    if (!res.body) {
      return reject(new Error("No body to pipe"));
    }

    const targetPath = path.resolve("./bin");
    debug("targetPath:", targetPath);

    /**
     * A different unpacking strategy will be needed for every extension, so
     * account for them all here.
     *
     * Bit messy having them inline, but all of this is only running once, so
     * it's fine to keep them here and in context.
     */
    const strategies: Record<KnownExtension, () => void> = {
      [KnownExtension.Tar]: () => {
        debug("unpacking using tar strategy");
        const untar = tar.extract({ cwd: targetPath });

        untar.on("error", reject);
        untar.on("end", () => resolve());

        res.body?.pipe(untar);
      },
      [KnownExtension.Zip]: () => {
        debug("unpacking using zip strategy");
        res
          .buffer()
          .then((buffer) => {
            const zip = new AdmZip(buffer);
            zip.extractAllTo(targetPath, true);
            resolve();
          })
          .catch(reject);
      },
    };

    /**
     * Find the correct strategy for us to use for unpacking the binary.
     */
    const [, strategy] = Object.entries(strategies).find(([knownExt]) => {
      return originalUrl.pathname.endsWith(knownExt);
    }) || [, null];

    if (!strategy) {
      return reject(new Error("Unsupported extension"));
    }

    strategy();
  });
}

(async () => {
  rootDebug("postinstall started");

  /**
   * Allow skipping this step for builds and local dev.
   */
  if (process.env.SKIP_POSTINSTALL) {
    rootDebug("SKIP_POSTINSTALL was defined; skipping postinstall");
    process.exit(0);
  }

  try {
    const binaryUrl = await getBinaryUrl();
    const req = await downloadBinary(binaryUrl);
    await pipeBinaryToInstallLocation(req, binaryUrl);
    rootDebug("postinstall complete");
  } catch (err) {
    console.error(err);
    process.exit(1);
  }
})();
