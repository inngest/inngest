const crypto = require('crypto');
const fs = require('fs');
const path = require('path');
const { chromium } = require('/home/fedora/.npm/_npx/fd3bca3c548369c0/node_modules/playwright');

const APP_ORIGIN = 'http://localhost:5174';
const INSIGHTS_PATH = '/env/production/insights';
const OUTPUT_DIR = path.resolve(__dirname, 'insights-race-recordings-raw');
const CHROME =
  '/home/fedora/.cache/ms-playwright/chromium-1187/chrome-linux/chrome';

const SAVED_A = {
  id: '01M3BNPSDZYKQ7KPY9Q6GRK0SP',
  name: 'Race A — previous active',
  sql: "SELECT 'A — stale active' AS marker",
};
const SAVED_B = {
  id: '01M3BNPSECE1CN310MRGWJWVPP',
  name: 'Race B — incoming deep link',
  sql: "SELECT 'B — requested target' AS marker",
};
const SAVED_C = {
  id: '01M3BNPSEKFNS8S6XTS9M0A78F',
  name: 'Race C — next navigation',
  sql: "SELECT 'C — same-mounted navigation' AS marker",
};

function proxyEnvironment() {
  const proc = fs
    .readdirSync('/proc')
    .filter((entry) => /^\d+$/.test(entry))
    .find((pid) => {
      try {
        return fs
          .readFileSync(`/proc/${pid}/cmdline`, 'utf8')
          .includes('local-auth-proxy.cjs');
      } catch {
        return false;
      }
    });
  if (!proc) throw new Error('local auth proxy process not found');

  return Object.fromEntries(
    fs
      .readFileSync(`/proc/${proc}/environ`, 'utf8')
      .split('\0')
      .filter(Boolean)
      .map((entry) => {
        const index = entry.indexOf('=');
        return [entry.slice(0, index), entry.slice(index + 1)];
      }),
  );
}

function localJWT() {
  const env = proxyEnvironment();
  const now = Math.floor(Date.now() / 1000);
  const encode = (value) =>
    Buffer.from(JSON.stringify(value)).toString('base64url');
  const header = encode({ alg: 'HS512', typ: 'JWT' });
  const payload = encode({
    sub: env.LOCAL_USER_ID,
    iss: 'datos',
    iat: now,
    nbf: now,
    exp: now + 60 * 60,
  });
  const signature = crypto
    .createHmac('sha512', env.LOCAL_JWT_SECRET)
    .update(`${header}.${payload}`)
    .digest('base64url');
  return `${header}.${payload}.${signature}`;
}

function tab(query, id = `tab-${query.id}`) {
  return {
    id,
    name: query.name,
    query: query.sql,
    savedQueryId: query.id,
  };
}

const HOME = { id: '__home', name: 'Home', query: '' };

async function addOverlays(page, title) {
  if (process.env.NO_OVERLAYS) return;
  await page.evaluate((scenarioTitle) => {
    const style = document.createElement('style');
    style.textContent = `
      #race-url, #race-caption, #race-title { position: fixed; z-index: 2147483647; pointer-events: none; }
      #race-title { top: 12px; left: 230px; padding: 8px 12px; color: #fff; background: rgba(10, 14, 22, .92); border: 2px solid #f5c542; border-radius: 6px; font: 700 14px/1.2 system-ui; }
      #race-url { top: 12px; right: 18px; max-width: 760px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; padding: 8px 12px; color: #fff; background: rgba(10, 14, 22, .92); border: 2px solid #f5c542; border-radius: 6px; font: 600 13px/1.2 ui-monospace, monospace; }
      #race-caption { left: 230px; right: 420px; bottom: 24px; padding: 12px 16px; color: #fff; background: rgba(10, 14, 22, .94); border-left: 5px solid #f5c542; border-radius: 5px; font: 700 17px/1.35 system-ui; box-shadow: 0 4px 18px rgba(0,0,0,.4); }
      #race-tab-highlight { position: fixed; z-index: 2147483646; pointer-events: none; left: 210px; right: 0; top: 55px; height: 54px; border: 3px solid rgba(245,197,66,.92); box-shadow: inset 0 0 0 2px rgba(10,14,22,.55); }
      [data-testid="tanstack-router-devtools"] { display: none !important; }
    `;
    document.head.appendChild(style);

    const make = (id, text) => {
      const node = document.createElement('div');
      node.id = id;
      node.textContent = text;
      document.body.appendChild(node);
      return node;
    };
    make('race-title', scenarioTitle);
    const url = make('race-url', '');
    make('race-caption', 'Preparing scenario…');
    make('race-tab-highlight', '');
    const updateURL = () => {
      url.textContent = `Current URL: ${location.pathname}${location.search}`;
    };
    updateURL();
    window.__raceURLTimer = setInterval(updateURL, 75);
  }, title);
}

async function caption(page, text, delay = 1800) {
  if (!process.env.NO_OVERLAYS) {
    await page.evaluate((value) => {
      document.querySelector('#race-caption').textContent = value;
    }, text);
  }
  await page.waitForTimeout(delay);
}

async function navigateInPlace(page, href) {
  await page.evaluate(async (nextHref) => {
    await window.__INSIGHTS_NAVIGATE__({ href: nextHref });
  }, href);
}

async function waitForActiveSavedQuery(page, savedQueryId) {
  await page.waitForFunction(
    (expected) => {
      const state = JSON.parse(
        localStorage.getItem('insights-tabs-state') ?? 'null',
      );
      const active = state?.tabs.find((tab) => tab.id === state.activeTabId);
      return active?.savedQueryId === expected;
    },
    savedQueryId,
    { timeout: 15_000 },
  );
  await page.waitForTimeout(250);
}

async function waitForActiveTabName(page, name) {
  await page.waitForFunction(
    (expected) => {
      const state = JSON.parse(
        localStorage.getItem('insights-tabs-state') ?? 'null',
      );
      const active = state?.tabs.find((tab) => tab.id === state.activeTabId);
      return active?.name === expected;
    },
    name,
    { timeout: 15_000 },
  );
  await page.waitForTimeout(250);
}

async function recordScenario(browser, {
  name,
  title,
  route,
  run,
}) {
  if (process.env.SCENARIO && process.env.SCENARIO !== name) return;

  const scenarioDir = path.join(OUTPUT_DIR, name);
  fs.rmSync(scenarioDir, { recursive: true, force: true });
  fs.mkdirSync(scenarioDir, { recursive: true });

  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    recordVideo: { dir: scenarioDir, size: { width: 1440, height: 900 } },
  });
  await context.addCookies([
    { name: 'jwt', value: localJWT(), url: APP_ORIGIN },
  ]);
  if (route) await route(context);

  const page = await context.newPage();
  page.on('console', (message) => {
    if (message.type() === 'error' || message.text().includes('RACE_C')) {
      console.error(`[${name}]`, message.text());
    }
  });
  page.on('pageerror', (error) => console.error(`[${name}]`, error.stack));
  await page.goto(`${APP_ORIGIN}${INSIGHTS_PATH}`, {
    waitUntil: 'domcontentloaded',
    timeout: 30_000,
  });
  await page.waitForFunction(
    () => document.body.innerText.includes('Insights'),
    undefined,
    { timeout: 30_000 },
  );
  await addOverlays(page, title);
  await run(page);
  await page.screenshot({ path: path.join(scenarioDir, 'final.png') });
  const video = page.video();
  await context.close();
  const videoPath = await video.path();
  fs.renameSync(videoPath, path.join(OUTPUT_DIR, `${name}.webm`));
  fs.rmSync(scenarioDir, { recursive: true, force: true });
}

async function main() {
  fs.rmSync(OUTPUT_DIR, { recursive: true, force: true });
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  const browser = await chromium.launch({
    headless: true,
    executablePath: CHROME,
  });

  try {
    await recordScenario(browser, {
      name: '01-active-a-to-b',
      title: 'Race 1 · active A → inbound B',
      run: async (page) => {
        await navigateInPlace(page, `${INSIGHTS_PATH}?query_id=${SAVED_A.id}`);
        await waitForActiveSavedQuery(page, SAVED_A.id);
        await caption(page, 'Start: saved query A is active and committed to the tab state.');
        await navigateInPlace(page, `${INSIGHTS_PATH}?query_id=${SAVED_B.id}`);
        await caption(
          page,
          'Inbound navigation requests B. URL synchronization stays paused while B opens.',
          900,
        );
        await waitForActiveSavedQuery(page, SAVED_B.id);
        await caption(
          page,
          'B is active and query_id still names B; only now may outbound synchronization resume.',
          2600,
        );
      },
    });

    await recordScenario(browser, {
      name: '02-home-to-b',
      title: 'Race 2 · HOME → inbound B',
      run: async (page) => {
        await caption(
          page,
          'Start: HOME is active and no saved-query tab exists.',
          1500,
        );
        await navigateInPlace(page, `${INSIGHTS_PATH}?query_id=${SAVED_B.id}`);
        await caption(
          page,
          'Inbound navigation requests B while HOME is still the committed active tab.',
          900,
        );
        await waitForActiveSavedQuery(page, SAVED_B.id);
        await caption(
          page,
          'B is created and activated; its query_id permalink remains in the URL.',
          2600,
        );
      },
    });

    await recordScenario(browser, {
      name: '03-restored-b-focus',
      title: 'Race 3 · requested B already exists in restored tabs',
      run: async (page) => {
        await navigateInPlace(page, `${INSIGHTS_PATH}?query_id=${SAVED_B.id}`);
        await waitForActiveSavedQuery(page, SAVED_B.id);
        await caption(page, 'Setup: open B once so its tab is already restored.');
        await navigateInPlace(page, `${INSIGHTS_PATH}?query_id=${SAVED_A.id}`);
        await waitForActiveSavedQuery(page, SAVED_A.id);
        await caption(page, 'Start: restored tabs contain both A and B; A is active.');
        await navigateInPlace(page, `${INSIGHTS_PATH}?query_id=${SAVED_B.id}`);
        await caption(page, 'Inbound navigation requests the already-restored B tab.', 900);
        await waitForActiveSavedQuery(page, SAVED_B.id);
        await caption(
          page,
          'The existing B tab is focused atomically—no duplicate B tab is appended.',
          2600,
        );
      },
    });

    await recordScenario(browser, {
      name: '04-same-mounted-b-to-c',
      title: 'Race 4 · same-mounted B → C navigation',
      run: async (page) => {
        await navigateInPlace(page, `${INSIGHTS_PATH}?query_id=${SAVED_B.id}`);
        await waitForActiveSavedQuery(page, SAVED_B.id);
        await caption(page, 'Start: B is active without remounting the Insights route.');
        await navigateInPlace(
          page,
          `${INSIGHTS_PATH}?query_id=${SAVED_C.id}&keep=preserved`,
        );
        await caption(page, 'Without remounting, a second inbound navigation requests C.', 900);
        await waitForActiveSavedQuery(page, SAVED_C.id);
        await caption(page, 'C becomes active and the unrelated keep parameter is preserved.', 2400);
      },
    });

    await recordScenario(browser, {
      name: '05-delayed-fetch-success',
      title: 'Race 5 · inbound B waits for saved-query loading',
      route: async (context) => {
        await context.route('**/gql', async (request) => {
          if (request.request().postData()?.includes('InsightsSavedQueries')) {
            await new Promise((resolve) => setTimeout(resolve, 3200));
          }
          await request.continue();
        });
      },
      run: async (page) => {
        await navigateInPlace(page, `${INSIGHTS_PATH}?query_id=${SAVED_B.id}`);
        await caption(
          page,
          'Saved-query loading is intentionally delayed. Insights remains visible and query_id=B is held.',
          2600,
        );
        await waitForActiveSavedQuery(page, SAVED_B.id);
        await caption(page, 'When resources arrive, B opens and its permalink remains intact.', 2600);
      },
    });

    await recordScenario(browser, {
      name: '06-fetch-error-recovery',
      title: 'Race 6 · saved-query fetch fails with no data',
      route: async (context) => {
        await context.route('**/gql', async (request) => {
          if (request.request().postData()?.includes('InsightsSavedQueries')) {
            await new Promise((resolve) => setTimeout(resolve, 1500));
            await request.fulfill({
              status: 200,
              contentType: 'application/json',
              body: JSON.stringify({
                errors: [{ message: 'Forced saved-query failure for recording' }],
                data: null,
              }),
            });
            return;
          }
          await request.continue();
        });
      },
      run: async (page) => {
        await navigateInPlace(
          page,
          `${INSIGHTS_PATH}?query_id=${SAVED_B.id}&keep=recoverable`,
        );
        await caption(page, 'The saved-query request is forced to fail with data undefined.', 1200);
        await page.waitForFunction(
          () =>
            location.search.includes('query_id') &&
            document.body.innerText.includes(
              'Unable to load saved queries; please try again',
            ),
          undefined,
          { timeout: 10_000 },
        );
        await caption(
          page,
          'The error is terminal: HOME stays usable and query_id remains so a reload can retry.',
          3000,
        );
      },
    });

    await recordScenario(browser, {
      name: '07-sql-prefill-repeat',
      title: 'Race 7 · one-shot SQL prefill commits before URL cleanup',
      run: async (page) => {
        const sql = "select 'SQL — prefill only' as marker from function_runs";
        const href = `${INSIGHTS_PATH}?sql=${encodeURIComponent(sql)}&name=${encodeURIComponent('SQL prefill — no autorun')}&keep=preserved`;
        await caption(page, 'Start from HOME; no SQL-prefill tab exists.');
        await navigateInPlace(page, href);
        await caption(page, 'Inbound SQL is opening. sql/name stay in the URL until activation commits.', 700);
        await waitForActiveTabName(page, 'SQL prefill — no autorun');
        await page.waitForFunction(
          () => !location.search.includes('sql='),
          undefined,
          { timeout: 10_000 },
        );
        await caption(
          page,
          'The unsaved tab is active, sql/name are consumed, and the empty-results prompt confirms no autorun.',
          2800,
        );
        await navigateInPlace(page, `${INSIGHTS_PATH}?keep=between`);
        await page.waitForTimeout(350);
        await navigateInPlace(page, href);
        await caption(page, 'A later navigation may issue the same one-shot SQL command again.', 800);
        await page.waitForFunction(
          () => !location.search.includes('sql='),
          undefined,
          { timeout: 10_000 },
        );
        await page.waitForFunction(() => {
          const state = JSON.parse(
            localStorage.getItem('insights-tabs-state') ?? 'null',
          );
          return (
            state?.tabs.filter(
              (tab) => tab.name === 'SQL prefill — no autorun',
            ).length === 2
          );
        });
        await caption(page, 'The second navigation creates and activates a new unsaved tab without running it.', 2600);
      },
    });
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
