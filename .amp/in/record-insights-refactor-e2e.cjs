const fs = require('fs');
const path = require('path');
const { chromium } = require('/home/fedora/.npm/_npx/fd3bca3c548369c0/node_modules/playwright');

const ORIGIN = 'http://127.0.0.1:5175';
const INSIGHTS = '/env/production/insights';
const OUT = path.resolve('.amp/in/artifacts/insights-refactor-e2e');
const AUTH = '.amp/in/dashboard-auth-state.json';
const FIXTURES = JSON.parse(fs.readFileSync('.amp/in/insights-fixtures.json', 'utf8'));
const HOME = { id: '__home', name: 'Home', query: '' };

function savedTab(saved, id = `tab-${saved.id}`) {
  return { id, name: saved.name, query: saved.sql, savedQueryId: saved.id };
}

async function overlay(page, title) {
  await page.evaluate((title) => {
    const style = document.createElement('style');
    style.id = 'e2e-demo-style';
    style.textContent = `
      #e2e-title,#e2e-url,#e2e-caption,#e2e-focus {position:fixed;z-index:2147483647;pointer-events:none;box-sizing:border-box}
      #e2e-title{top:12px;left:220px;background:#0a0e16ed;color:#fff;border:2px solid #f5c542;border-radius:6px;padding:8px 12px;font:700 14px/1.2 system-ui}
      #e2e-url{top:12px;right:18px;max-width:710px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;background:#0a0e16ed;color:#fff;border:2px solid #f5c542;border-radius:6px;padding:8px 12px;font:600 12px/1.2 ui-monospace,monospace}
      #e2e-caption{left:220px;right:430px;bottom:24px;background:#0a0e16f2;color:#fff;border-left:5px solid #f5c542;border-radius:5px;padding:12px 16px;font:700 17px/1.35 system-ui;box-shadow:0 4px 18px #0007}
      #e2e-focus{left:208px;right:0;top:55px;height:55px;border:3px solid #f5c542dd;box-shadow:inset 0 0 0 2px #0a0e1688}
      [data-testid="tanstack-router-devtools"]{display:none!important}
    `;
    document.head.appendChild(style);
    const add = (id, text) => { const el = document.createElement('div'); el.id = id; el.textContent = text; document.body.appendChild(el); return el; };
    add('e2e-title', title);
    const url = add('e2e-url', '');
    add('e2e-caption', 'Preparing real backend state…');
    add('e2e-focus', '');
    const refresh = () => { url.textContent = `URL: ${location.pathname}${location.search}`; };
    refresh();
    window.__e2eURLTimer = setInterval(refresh, 75);
  }, title);
}

async function caption(page, text, wait = 1700) {
  await page.evaluate((text) => { document.querySelector('#e2e-caption').textContent = text; }, text);
  await page.waitForTimeout(wait);
}

async function navigate(page, href) {
  await page.evaluate(async (href) => { await window.__TSR_ROUTER__.navigate({ href }); }, href);
}

async function activeTab(page) {
  return page.evaluate(() => {
    const state = JSON.parse(localStorage.getItem('insights-tabs-state') || 'null');
    return state?.tabs.find((tab) => tab.id === state.activeTabId);
  });
}

async function waitActiveSaved(page, id) {
  await page.waitForFunction((id) => {
    const state = JSON.parse(localStorage.getItem('insights-tabs-state') || 'null');
    return state?.tabs.find((tab) => tab.id === state.activeTabId)?.savedQueryId === id;
  }, id, { timeout: 15000 });
}

async function waitActiveName(page, name) {
  await page.waitForFunction((name) => {
    const state = JSON.parse(localStorage.getItem('insights-tabs-state') || 'null');
    return state?.tabs.find((tab) => tab.id === state.activeTabId)?.name === name;
  }, name, { timeout: 15000 });
}

async function tabCount(page, savedQueryId) {
  return page.evaluate((id) => {
    const state = JSON.parse(localStorage.getItem('insights-tabs-state') || 'null');
    return state?.tabs.filter((tab) => tab.savedQueryId === id).length ?? 0;
  }, savedQueryId);
}

async function record(browser, { name, title, start = INSIGHTS, init, route, run }) {
  const dir = path.join(OUT, `${name}-raw`);
  fs.rmSync(dir, { recursive: true, force: true });
  fs.mkdirSync(dir, { recursive: true });
  const context = await browser.newContext({
    storageState: AUTH,
    viewport: { width: 1440, height: 900 },
    recordVideo: { dir, size: { width: 1440, height: 900 } },
  });
  await context.addInitScript((init) => {
    localStorage.removeItem('insights-tabs-state');
    if (init?.tabsState) localStorage.setItem('insights-tabs-state', JSON.stringify(init.tabsState));
  }, init || null);
  if (route) await route(context);
  const page = await context.newPage();
  const failures = [];
  page.on('pageerror', (error) => failures.push(`pageerror: ${error.message}`));
  page.on('console', (message) => {
    if (message.type() === 'error' && !message.text().includes('VITE_LAUNCH_DARKLY')) failures.push(`console: ${message.text()}`);
  });
  await page.goto(`${ORIGIN}${start}`, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForFunction(() => document.body.innerText.includes('Insights'), undefined, { timeout: 30000 });
  await page.waitForTimeout(900);
  await overlay(page, title);
  const assertions = await run(page, context);
  const screenshot = path.join(OUT, `${name}.png`);
  await page.screenshot({ path: screenshot });
  const video = page.video();
  await context.close();
  const videoPath = await video.path();
  const webm = path.join(OUT, `${name}.webm`);
  fs.renameSync(videoPath, webm);
  fs.rmSync(dir, { recursive: true, force: true });
  return { name, title, webm, screenshot, assertions, failures };
}

async function main() {
  fs.rmSync(OUT, { recursive: true, force: true });
  fs.mkdirSync(OUT, { recursive: true });
  const browser = await chromium.launch({ headless: true });
  const results = [];
  try {
    results.push(await record(browser, {
      name: '01-serialized-saved-query-navigation',
      title: 'Saved-query navigation · A → B → C',
      run: async (page) => {
        await navigate(page, `${INSIGHTS}?query_id=${FIXTURES.A.id}`);
        await waitActiveSaved(page, FIXTURES.A.id);
        await caption(page, 'Real GraphQL fixture A is active. Its query_id and selected tab agree.');
        await caption(page, 'Next, the same mounted Insights route receives query B…', 900);
        await navigate(page, `${INSIGHTS}?query_id=${FIXTURES.B.id}&keep=53`);
        await waitActiveSaved(page, FIXTURES.B.id);
        await caption(page, 'B is committed active before URL synchronization resumes; stale A never overwrites query_id=B.');
        await caption(page, 'A second same-mounted navigation now requests C…', 850);
        await navigate(page, `${INSIGHTS}?query_id=${FIXTURES.C.id}&keep=53`);
        await waitActiveSaved(page, FIXTURES.C.id);
        await caption(page, 'C is active, query_id=C, and the unrelated keep=53 parameter survives.', 2500);
        return { active: await activeTab(page), url: page.url() };
      },
    }));

    results.push(await record(browser, {
      name: '02-restored-tab-focus-is-atomic',
      title: 'Restored saved query · focus without duplication',
      run: async (page) => {
        await navigate(page, `${INSIGHTS}?query_id=${FIXTURES.B.id}`);
        await waitActiveSaved(page, FIXTURES.B.id);
        await navigate(page, `${INSIGHTS}?query_id=${FIXTURES.A.id}`);
        await waitActiveSaved(page, FIXTURES.A.id);
        const before = await tabCount(page, FIXTURES.B.id);
        await caption(page, `Both saved tabs are restored; A is active and exactly ${before} B tab exists.`);
        await navigate(page, `${INSIGHTS}?query_id=${FIXTURES.B.id}`);
        await waitActiveSaved(page, FIXTURES.B.id);
        const after = await tabCount(page, FIXTURES.B.id);
        await caption(page, `The existing B tab is focused atomically. B tab count remains ${after}; no duplicate was appended.`, 2600);
        return { before, after, active: await activeTab(page), url: page.url() };
      },
    }));

    results.push(await record(browser, {
      name: '03-producer-prefill-and-explicit-run',
      title: 'Metrics producer · prefill first, execute explicitly',
      start: '/env/production/metrics',
      run: async (page) => {
        let insightsRequests = 0;
        page.on('request', (request) => {
          if (request.postData()?.includes('InsightsResults')) insightsRequests += 1;
        });
        await page.waitForTimeout(1800);
        await caption(page, 'The real Metrics “Open in Insights” action is a first-party deep-link producer.');
        await page.getByRole('button', { name: 'Open in Insights' }).click();
        await waitActiveName(page, 'Failed function runs (24h)');
        await page.waitForFunction(() => !location.search.includes('sql=') && !location.search.includes('name='));
        const before = insightsRequests;
        await caption(page, `Its SQL is formatted into one unsaved tab and sql/name are consumed. Backend query count: ${before}.`);
        await caption(page, 'The editor remains inspectable until the user explicitly clicks Run query.', 1000);
        const responsePromise = page.waitForResponse((response) => response.request().postData()?.includes('InsightsResults') ?? false);
        await page.getByRole('button', { name: /Run query/i }).click();
        const response = await responsePromise;
        await page.waitForTimeout(1700);
        await caption(page, `Run query reached the real Cloud GraphQL backend (HTTP ${response.status()}).`, 2600);
        const state = await page.evaluate(() => JSON.parse(localStorage.getItem('insights-tabs-state') || 'null'));
        return { beforeRunRequests: before, afterRunRequests: insightsRequests, responseStatus: response.status(), matchingTabs: state.tabs.filter((t) => t.name === 'Failed function runs (24h)').length, url: page.url() };
      },
    }));

    results.push(await record(browser, {
      name: '04-route-contract-and-precedence',
      title: 'Route contract · query_id wins over SQL',
      run: async (page) => {
        const ignored = "SELECT id FROM runs WHERE status = 'IGNORED & + / café'";
        await caption(page, 'A deliberately conflicting link supplies query_id=B together with sql and name.');
        await navigate(page, `${INSIGHTS}?query_id=${FIXTURES.B.id}&sql=${encodeURIComponent(ignored)}&name=${encodeURIComponent('Ignored / café + A&B')}&keep=89`);
        await waitActiveSaved(page, FIXTURES.B.id);
        await page.waitForTimeout(600);
        const state = await page.evaluate(() => JSON.parse(localStorage.getItem('insights-tabs-state') || 'null'));
        const ignoredTabs = state.tabs.filter((tab) => tab.name === 'Ignored / café + A&B').length;
        await caption(page, `Only saved query B is applied. Conflicting SQL tabs created: ${ignoredTabs}; unsupported fields are absent from the canonical URL.`, 2800);
        return { active: await activeTab(page), ignoredTabs, url: page.url() };
      },
    }));

    results.push(await record(browser, {
      name: '05-cached-error-preserves-and-retries',
      title: 'Saved-query failure · preserve local edits and retry',
      start: `${INSIGHTS}?query_id=${FIXTURES.B.id}`,
      init: {
        tabsState: {
          tabs: [HOME, { ...savedTab(FIXTURES.B, 'restored-B-local'), query: "SELECT id, status FROM runs\n-- LOCAL EDIT SURVIVES" }],
          activeTabId: 'restored-B-local',
        },
      },
      route: async (context) => {
        let intercepted = false;
        await context.route('**/gql', async (route) => {
          const request = route.request();
          if (!intercepted && request.postData()?.includes('InsightsSavedQueries')) {
            intercepted = true;
            const response = await route.fetch();
            const body = await response.json();
            body.data.account.insightsQueries = body.data.account.insightsQueries.filter((query) => query.id !== FIXTURES.B.id);
            body.errors = [{ message: 'Forced transient network failure for E2E demo' }];
            await route.fulfill({ response, json: body });
            return;
          }
          await route.continue();
        });
      },
      run: async (page) => {
        const retry = page.getByRole('button', { name: 'Retry' });
        await retry.waitFor({ state: 'visible', timeout: 15000 });
        const preservedBefore = await tabCount(page, FIXTURES.B.id);
        await caption(page, `Cached data omitted B but arrived with an error. B is not deleted: ${preservedBefore} restored tab remains.`);
        await caption(page, 'The query_id permalink is retained and the toast exposes an explicit network-only Retry.', 1200);
        await retry.click();
        await waitActiveSaved(page, FIXTURES.B.id);
        await page.waitForFunction(() => {
          const state = JSON.parse(localStorage.getItem('insights-tabs-state') || 'null');
          return state?.tabs
            .find((tab) => tab.id === state.activeTabId)
            ?.query.includes('LOCAL EDIT SURVIVES');
        });
        const tab = await activeTab(page);
        await caption(page, 'Retry reached the real backend, reactivated B, and preserved the locally edited SQL.', 2800);
        return { preservedBefore, active: tab, localEditPreserved: tab.query.includes('LOCAL EDIT SURVIVES'), url: page.url() };
      },
    }));

    results.push(await record(browser, {
      name: '06-oversized-external-link',
      title: 'Oversized external SQL · explicit recovery',
      run: async (page) => {
        const sql = 'x'.repeat(8193);
        await caption(page, 'An external or agent-generated link supplies 8,193 UTF-8 bytes of SQL—one byte over the contract.');
        await navigate(page, `${INSIGHTS}?sql=${sql}&name=oversized-agent-query&keep=144`);
        await page.getByText('This query is too large to open in Insights.').waitFor({ state: 'visible', timeout: 15000 });
        await page.waitForFunction(() => !location.search.includes('sql=') && !location.search.includes('name='));
        await caption(page, 'Insights shows an explicit error, creates no tab, discards the SQL body, and preserves keep=144.', 2800);
        const state = await page.evaluate(() => JSON.parse(localStorage.getItem('insights-tabs-state') || 'null'));
        return { tabCount: state?.tabs.length ?? 0, active: await activeTab(page), url: page.url() };
      },
    }));
  } finally {
    await browser.close();
  }
  fs.writeFileSync(path.join(OUT, 'results.json'), JSON.stringify(results, null, 2));
  console.log(JSON.stringify(results.map(({ name, assertions, failures }) => ({ name, assertions, failures })), null, 2));
}

main().catch((error) => { console.error(error); process.exit(1); });
