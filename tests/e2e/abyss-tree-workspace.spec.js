const { test, expect } = require('@playwright/test');

const runtimeErrors = new WeakMap();
test.beforeEach(async ({ page }) => {
  const errors = [];
  runtimeErrors.set(page, errors);
  page.on('pageerror', error => errors.push(error.message));
});
test.afterEach(async ({ page }) => {
  expect(runtimeErrors.get(page), 'Skill Web initialization and interactions must not throw').toEqual([]);
});

async function mockTreeAPI(page, options = {}) {
  const writes = [];
  const previewRequests = [];
  const preview = { fail: false, tokenTotal: options.tokenTotal ?? 37, free: options.free ?? false, holdNext: false, pending: [] };
  await page.route('**/api/abyss/tree/**', async route => {
    const request = route.request();
    const endpoint = new URL(request.url()).pathname.split('/').pop();
    const body = request.postDataJSON() || {};
    if (endpoint === 'plan_preview') {
      previewRequests.push(body);
      if (preview.fail) return route.fulfill({ json: { ok: false, error: 'Preview unavailable; retry before applying.' } });
      const ids = body.ids || options.resolvedIDs || [];
      const tokenTotal = preview.tokenTotal;
      if (preview.holdNext) {
        preview.holdNext = false;
        await new Promise(resolve => preview.pending.push({ body, release: resolve }));
      }
      return route.fulfill({ json: { ok: true, analysis: {
        ids, missing: [], added: ids, removed: [], current_cost: 0,
        planned_cost: ids.length, available_points: 1000, connected: true,
        max_depth_gate: 0, depth_record: 50, current_stats: {}, planned_stats: {},
        stat_delta: {}, current_pct: {}, planned_pct: {}, pct_delta: {}, node_costs: {},
        warnings: [], valid: true, layout_hash: 'e2e-workspace-layout', schema_version: 1,
      }, quote: { token_total: tokenTotal, free: preview.free } } });
    }
    if (endpoint === 'plan_draft') return route.fulfill({ json: { ok: true, drafts: {} } });
    writes.push({ endpoint, body });
    return route.fulfill({ json: { ok: false, error: 'Mutation captured by the browser test.' } });
  });
  return { writes, preview, previewRequests };
}

async function connectedNode(page) {
  return page.evaluate(() => {
    const node = NODES.find(candidate => candidate.id !== 0 && isAllocatable(candidate.id));
    if (!node) throw new Error('The fixture needs an available node connected to the root.');
    return { id: node.id, name: node.name };
  });
}

async function tapExpandedTarget(page, target) {
  await target.scrollIntoViewIfNeeded();
  const point = await target.evaluate(element => {
    const matrix = element.getScreenCTM();
    const centerX = Number(element.getAttribute('cx'));
    const centerY = Number(element.getAttribute('cy'));
    const radius = Number(element.getAttribute('r'));
    const stroke = Number.parseFloat(getComputedStyle(element).strokeWidth);
    for (const scale of [0.35, 0.2, 0.45]) {
      for (let step = 0; step < 36; step++) {
        const angle = step * Math.PI / 18;
        const candidate = new DOMPoint(centerX + (radius + stroke * scale) * Math.cos(angle), centerY + (radius + stroke * scale) * Math.sin(angle)).matrixTransform(matrix);
        if (document.elementFromPoint(candidate.x, candidate.y) === element) return { x: candidate.x, y: candidate.y };
      }
    }
    return null;
  });
  expect(point, 'The expanded touch ring needs an exposed, reachable point').not.toBeNull();
  await page.touchscreen.tap(point.x, point.y);
}

test('planning keyboard activation changes only the draft', async ({ page }) => {
  const api = await mockTreeAPI(page);
  await page.goto('/abyss/tree');
  const node = await connectedNode(page);
  await page.locator('#treePlanToggle').click();
  const target = page.locator(`#treeSvg .tn[data-id="${node.id}"]`);
  await target.focus();
  await target.press('Enter');
  expect.soft(api.writes, 'Planning must not call a persistent mutation endpoint').toEqual([]);
  await expect(page.locator('#treePlanPlannedCount')).toHaveText('1');
  expect(api.writes).toEqual([]);
  await target.press('Space');
  await expect(page.locator('#treePlanPlannedCount')).toHaveText('0');
  expect(api.writes).toEqual([]);
});

test('ordinary node clicks inspect before an explicit allocation', async ({ page }) => {
  const api = await mockTreeAPI(page);
  await page.goto('/abyss/tree');
  const node = await connectedNode(page);
  const target = page.locator(`#treeSvg .tn[data-id="${node.id}"]`);
  await target.click();
  await expect(page.locator('#treeInspectorTitle')).toHaveText(node.name);
  expect(api.writes).toEqual([]);
  const allocate = page.locator('#treeInspectorAllocate');
  await expect(allocate).toBeVisible();
  await expect(allocate).toBeEnabled();
  await allocate.click();
  await expect.poll(() => api.writes).toEqual([{ endpoint: 'allocate', body: { node_id: node.id } }]);
});

test('the graph has one keyboard entry point and filtered nodes are skipped', async ({ page }) => {
  await mockTreeAPI(page);
  await page.goto('/abyss/tree');
  const entry = page.locator('#treeSvg .tn[tabindex="0"]');
  await expect(entry).toHaveCount(1);
  await entry.focus();
  const originalID = await entry.getAttribute('data-id');
  await entry.press('ArrowRight');
  await expect(entry).toHaveCount(1);
  await expect(entry).toBeFocused();
  expect(await entry.getAttribute('data-id')).not.toBe(originalID);

  await page.locator('#treeSearch').fill('Limit Break');
  await expect(page.locator('#treeSvg .tn[data-id]:not(.tree-nav-hidden)')).toHaveCount(1);
  await expect(page.locator('#treeSvg .tn.tree-nav-hidden[tabindex="0"]')).toHaveCount(0);
  await expect(entry).toHaveCount(1);
  await entry.focus();
  await entry.press('ArrowRight');
  await expect(entry).toBeFocused();
  await expect(entry).not.toHaveClass(/tree-nav-hidden/);
});

test('the graph leads the workspace and the compact inspector fits desktop and mobile', async ({ page }) => {
  await mockTreeAPI(page);
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto('/abyss/tree');
  const stage = page.locator('#treeStage');
  const inspector = page.locator('.tree-inspector');
  const desktopGraph = await stage.boundingBox();
  const desktopInspector = await inspector.boundingBox();
  expect(desktopGraph.y).toBeLessThan(650);
  expect(desktopGraph.width).toBeGreaterThan(720);
  expect(desktopInspector.x).toBeGreaterThanOrEqual(desktopGraph.x + desktopGraph.width - 1);
  expect(desktopInspector.width).toBeLessThanOrEqual(380);
  expect(Math.abs(desktopInspector.y - desktopGraph.y)).toBeLessThan(100);

  await page.setViewportSize({ width: 390, height: 844 });
  const mobileGraph = await stage.boundingBox();
  expect(mobileGraph.y).toBeLessThan(650);
  expect(mobileGraph.width).toBeGreaterThan(300);
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1)).toBe(true);
  await page.locator('#treePlanToggle').click();
  for (const id of ['treePlanToggle', 'treePlanCommit']) {
    await expect(page.locator(`#${id}`)).toBeVisible();
    const bounds = await page.locator(`#${id}`).boundingBox();
    expect(bounds.height, `${id} needs a touch-sized target`).toBeGreaterThanOrEqual(44);
  }
});

for (const quote of [{ tokenTotal: 37, free: false }, { tokenTotal: 0, free: true }]) {
  test(`plan confirmation quotes ${quote.tokenTotal} tokens before any write`, async ({ page }) => {
    const api = await mockTreeAPI(page, quote);
    await page.goto('/abyss/tree');
    const node = await connectedNode(page);
    await page.locator('#treePlanToggle').click();
    await page.locator(`#treeSvg .tn[data-id="${node.id}"]`).press('Enter');
    await expect(page.locator('#treePlanPlannedCount')).toHaveText('1');
    await page.locator('#treePlanCommit').click();
    const modal = page.locator('#sharedModalCard');
    await expect(modal).toBeVisible();
    await expect(modal).toContainText(new RegExp(`${quote.tokenTotal}\\s*tokens?`, 'i'));
    await expect(modal).toContainText(/1\s+(?:nodes?\s+)?added|add(?:ed)?\s*:?\s*1/i);
    await expect(modal).toContainText(/0\s+(?:nodes?\s+)?removed|remove(?:d)?\s*:?\s*0/i);
    if (quote.free) await expect(modal).toContainText(/free/i);
    expect(api.writes).toEqual([]);
    await page.locator('#modalCancelBtn').click();
    expect(api.writes).toEqual([]);
    await page.locator('#treePlanCommit').click();
    await expect(modal).toBeVisible();
    await page.locator('#modalOkBtn').click();
    await expect.poll(() => api.writes).toEqual([{
      endpoint: 'build_import', body: { ids: [node.id], max_tokens: quote.tokenTotal },
    }]);
  });
}

test('a failed refresh cannot reuse an earlier valid preview to commit', async ({ page }) => {
  const api = await mockTreeAPI(page);
  await page.goto('/abyss/tree');
  const node = await connectedNode(page);
  await page.locator('#treePlanToggle').click();
  await page.locator(`#treeSvg .tn[data-id="${node.id}"]`).press('Enter');
  await expect(page.locator('#treePlanPlannedCount')).toHaveText('1');
  await expect(page.locator('#treePlanStatus')).toContainText(/valid|ready/i);
  api.preview.fail = true;
  await page.locator('#treePlanCommit').click();
  await expect(page.locator('#treePlanStatus')).toContainText(/unavailable|failed/i);
  await expect(page.locator('#sharedModal')).not.toHaveClass(/open/);
  expect(api.writes).toEqual([]);
});

test.describe('coarse pointer graph input', () => {
  test.use({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });

  test('expanded touch targets inspect normally and edit only the planning draft', async ({ page }) => {
    const api = await mockTreeAPI(page);
    await page.goto('/abyss/tree');
    expect(await page.evaluate(() => matchMedia('(pointer: coarse)').matches)).toBe(true);
    const node = await connectedNode(page);
    await page.locator('#treeSearch').fill(node.name);
    await page.locator('#treeDiscoveryDetails > summary').tap();
    await page.locator('#treeStateFilter').selectOption('available');
    await expect(page.locator('#treeResultCount')).toHaveText('1 result');
    await page.locator('#treeDiscoveryDetails > summary').tap();
    await page.locator('#treeFitResults').tap();
    const hit = page.locator(`#treeSvg .tree-touch-hit[data-touch-id="${node.id}"]`);
    await expect(hit).toHaveCount(1);
    // Find an exposed point outside the icon, then send a real touchscreen tap.
    await tapExpandedTarget(page, hit);
    await expect(page.locator('#treeInspectorTitle')).toHaveText(node.name);
    expect(api.writes).toEqual([]);
    await page.locator('#treePlanToggle').tap();
    await tapExpandedTarget(page, hit);
    await expect(page.locator('#treePlanPlannedCount')).toHaveText('1');
    await tapExpandedTarget(page, hit);
    await expect(page.locator('#treePlanPlannedCount')).toHaveText('0');
    expect(api.writes).toEqual([]);
  });
});

for (const source of ['preset', 'import']) {
  test(`${source} uses a server preview and the confirmed token ceiling`, async ({ page }) => {
    const resolvedIDs = [3];
    const api = await mockTreeAPI(page, { resolvedIDs });
    if (source === 'preset') {
      // Supply a populated saved slot in this test response; do not mutate app globals.
      await page.route('**/abyss/tree', async route => {
        const response = await route.fetch();
        const html = (await response.text()).replace(/var LOADOUTS = [^;]+;/, 'var LOADOUTS = {"1":1};');
        await route.fulfill({ response, body: html });
      });
    }
    await page.goto('/abyss/tree');
    const selector = source === 'preset' ? '[onclick="loadoutApply(1)"]' : '[onclick="buildImport()"]';
    const action = page.locator(selector);
    const disclosure = action.locator('xpath=ancestor::details[1]');
    if (await disclosure.count()) await disclosure.locator('summary').first().click();
    const code = Buffer.from(JSON.stringify({ v: 1, ids: [3] })).toString('base64');
    if (source === 'import') page.once('dialog', dialog => dialog.accept(code));
    await action.click();
    const expectedRequest = source === 'preset' ? { slot: 1 } : { code };
    await expect.poll(() => api.previewRequests.some(request => JSON.stringify(request) === JSON.stringify(expectedRequest))).toBe(true);
    const modal = page.locator('#sharedModalCard');
    await expect(modal).toBeVisible();
    await expect(modal).toContainText(/37\s*tokens/i);
    await expect(modal).toContainText(/1\s+nodes?\s+added/i);
    expect(api.writes).toEqual([]);
    await page.locator('#modalOkBtn').click();
    await expect.poll(() => api.writes).toEqual([{
      endpoint: 'build_import', body: { ids: resolvedIDs, max_tokens: 37 },
    }]);
  });
}

test('a late preview cannot overwrite the current draft or apply quote', async ({ page }) => {
  const api = await mockTreeAPI(page);
  await page.goto('/abyss/tree');
  await expect(page.locator('#treePlanApplyCost')).toContainText(/37\s*tokens/i);
  const node = await connectedNode(page);
  const nextNodeID = await page.evaluate(firstID => NODES.find(candidate => candidate.id !== firstID && candidate.id !== 0 && isAllocatable(candidate.id)).id, node.id);
  await page.locator('#treePlanToggle').click();
  await expect(page.locator('#treePlanStatus')).toContainText(/ready/i);
  const target = page.locator(`#treeSvg .tn[data-id="${node.id}"]`);
  api.preview.tokenTotal = 91;
  api.preview.holdNext = true;
  await target.press('Enter');
  await expect.poll(() => api.preview.pending.length).toBe(1);
  expect(api.preview.pending[0].body.ids).toEqual([node.id]);
  api.preview.tokenTotal = 37;
  const latestResponse = page.waitForResponse(response => response.url().endsWith('/plan_preview') && response.request().postDataJSON().ids.length === 2);
  await page.locator(`#treeSvg .tn[data-id="${nextNodeID}"]`).press('Enter');
  await (await latestResponse).finished();
  await expect(page.locator('#treePlanPlannedCount')).toHaveText('2');
  await expect(page.locator('#treePlanCost')).toContainText(/^2\s*\//);
  const oldResponse = page.waitForResponse(response => response.url().endsWith('/plan_preview') && response.request().postDataJSON().ids.length === 1);
  api.preview.pending[0].release();
  await (await oldResponse).finished();
  await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  expect(await page.locator('#treePlanCost').textContent()).toMatch(/^2\s*\//);
  await expect(page.locator('#treePlanPlannedCount')).toHaveText('2');
  await page.locator('#treePlanCommit').click();
  await expect(page.locator('#sharedModalCard')).toContainText(/37\s*tokens/i);
  await expect(page.locator('#sharedModalCard')).not.toContainText(/91\s*tokens/i);
  expect(api.writes).toEqual([]);
});
