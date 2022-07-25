import path from "path";
import request from "request";
import tar from "tar";
import unzipper from "unzipper";
import { URL } from "url";

/**
 * A map of Node's `process.arch` to Golang's `$GOARCH` equivalent.
 */
const archMap: Partial<Record<typeof process.arch, string>> = {
  arm64: "arm64",
  x64: "x86_64",
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
  const version = process.env.npm_package_version?.trim();

  if (!version) {
    throw new Error("Could not find package version to install binary");
  }

  const { arch, platform } = getArchPlatform();

  const targetUrl = new URL(
    `https://github.com/inngest/inngest/releases/download/v${version}/inngest_${version}_${platform.platform}_${arch}${platform.extension}`
  );

  return targetUrl;
}

/**
 * Return the platform and architecture included in the supposed binary name
 * from `$GOOS` and `$GOARCH`.
 *
 * @throws when the arch or platform are not supported by this script.
 */
function getArchPlatform() {
  const arch = archMap[process.arch];
  const platform = platformMap[process.platform];

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
): Promise<request.Request> {
  /**
   * Using `new Promise()` here so that we can use listeners when dealing with
   * the request in order to ensure it's valid without using the body.
   */
  return new Promise((resolve, reject) => {
    const req = request({ uri: url.href });

    req.on("response", (res) => {
      if (res.statusCode !== 200) {
        return reject(
          new Error(
            `Error downloading binary; invalid response status code: ${res.statusCode}`
          )
        );
      }

      resolve(req);
    });

    req.on("error", reject);
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
  req: request.Request,

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
  return new Promise<void>((resolve, reject) => {
    const onErr = (err: Error) => {
      console.error("Error unpackaining binary:", err);
      req.destroy();
      reject(err);
    };

    req.on("error", onErr);

    const targetPath = path.resolve("./bin");

    /**
     * A different unpacking strategy will be needed for every extension, so
     * account for them all here.
     *
     * Bit messy having them inline, but all of this is only running once, so
     * it's fine to keep them here and in context.
     */
    const strategies: Record<
      KnownExtension,
      (onErr: (err: Error) => void) => void
    > = {
      [KnownExtension.Tar]: () => {
        const untar = tar.extract({ cwd: targetPath });

        untar.on("error", onErr);
        untar.on("end", () => resolve());

        req.pipe(untar);
      },
      [KnownExtension.Zip]: () => {
        const unzip = unzipper.Extract({ path: targetPath });

        unzip.on("error", onErr);
        unzip.on("close", () => resolve());

        req.pipe(unzip);
      },
    };

    /**
     * Find the correct strategy for us to use for unpacking the binary.
     */
    const [, strategy] = Object.entries(strategies).find(([knownExt]) => {
      return originalUrl.pathname.endsWith(knownExt);
    }) || [, null];

    if (!strategy) {
      return onErr(new Error("Unsupported extension"));
    }

    strategy(onErr);
  });
}

(async () => {
  /**
   * Allow skipping this step for builds and local dev.
   */
  if (process.env.SKIP_POSTINSTALL) {
    process.exit(0);
  }

  try {
    const binaryUrl = await getBinaryUrl();
    const req = await downloadBinary(binaryUrl);
    await pipeBinaryToInstallLocation(req, binaryUrl);
  } catch (err) {
    /**
     * We assume that the thing that threw has already logged a specific error.
     */
    process.exit(1);
  }
})();
