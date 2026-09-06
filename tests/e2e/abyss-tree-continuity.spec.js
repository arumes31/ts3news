const { test, expect } = require('@playwright/test');

const runtimeErrors = new WeakMap();
test.beforeEach(async ({ page }) => {
  const errors = [];
  runtimeErrors.set(page, errors);
  page.on('pageerror', error => errors.push(error.message));
});
test.afterEach(async ({ page }) => {
  expect(runtimeErrors.get(page), 'Skill Web interactions must not throw').toEqual([]);
});

async function captureMutations(page) {
  const writes = [];
  await page.route('**/api/abyss/tree/**', async route => {
    const request = route.request();
    const endpoint = new URL(request.url()).pathname.split('/').pop();
    if (request.method() !== 'POST' || ['plan_preview', 'plan_draft'].includes(endpoint)) return route.continue();
    writes.push({ endpoint, body: request.postDataJSON() });
    return route.fulfill({ json: { ok: false, error: 'Mutation captured by the browser test.' } });
  });
  return writes;
}

async function connectedNode(page) {
  return page.evaluate(() => {
    const node = NODES.find(candidate => candidate.id !== 0 && isAllocatable(candidate.id));
    if (!node) throw new Error('The fixture needs a node connected to the root.');
    return { id: node.id, name: node.name };
  });
}

function namedResult(page, name) {
  return page.locator('#treeResultList button').filter({ hasText: name }).first();
}

async function tapGraphNode(page, id) {
  await page.locator('#treeStage').scrollIntoViewIfNeeded();
  const point = await page.locator(`#treeSvg .tn[data-id="${id}"]`).evaluate(element => {
    const center = new DOMPoint(Number(element.getAttribute('cx')), Number(element.getAttribute('cy'))).matrixTransform(element.getScreenCTM());
    return { x: center.x, y: center.y };
  });
  // Neighboring touch rings intentionally overlap. Send a real tap at the icon
  // center so the graph's nearest-node resolver handles the input as on a phone.
  await page.touchscreen.tap(point.x, point.y);
}

async function expectSelectedDetailsInViewport(page, name) {
  const title = page.locator('#treeInspectorTitle');
  await expect(title).toHaveText(name);
  await expect(title).toBeInViewport({ ratio: 1 });
  await expect(page.locator('#treeInspectorAllocate')).toBeInViewport({ ratio: 1 });
  const effects = page.locator('.tree-inspector-card').filter({ has: page.getByText('Effects', { exact: true }) });
  await expect(effects).toBeInViewport({ ratio: 1 });
}

test('keyboard selection owns bookmark, breadcrumb and recent history; hovering only previews', async ({ page }) => {
  await page.goto('/abyss/tree?node=3');
  const initial = page.locator('#treeSvg .tn[data-id="3"]');
  await initial.focus();
  await initial.press('ArrowRight');
  const selected = await page.locator('#treeSvg .tn[tabindex="0"]').getAttribute('data-id');
  expect(selected).not.toBe('3');
  const name = await page.evaluate(id => byId[id].name, selected);
  await expect(page.locator('#treeInspectorTitle')).toHaveText(name);
  await expect.soft(page.locator('#treeBookmark')).toContainText(name);
  await expect.soft(page.locator('#treeBreadcrumb button').last()).toHaveText(name);
  await expect.soft(page.locator('#treeRecent button').first()).toHaveText(name);

  // Keep the selected node in place while moving the real pointer to a nearby icon.
  const hoverPoint = await page.evaluate(selectedID => {
    for (const element of document.querySelectorAll('#treeSvg .tn[data-id]')) {
      if (element.getAttribute('data-id') === selectedID) continue;
      const bounds = element.getBoundingClientRect();
      const x = bounds.x + bounds.width / 2, y = bounds.y + bounds.height / 2;
      if (document.elementFromPoint(x, y) === element) return { x, y };
    }
    return null;
  }, selected);
  expect(hoverPoint, 'The fixture needs another exposed icon to exercise hover').not.toBeNull();
  await page.mouse.move(hoverPoint.x, hoverPoint.y);
  await expect.soft(page.locator('#treeInspectorTitle')).toHaveText(name);
  await expect.soft(page.locator('#treeBookmark')).toContainText(name);
  await expect.soft(page.locator('#treeRecent button').first()).toHaveText(name);
  await page.getByRole('button', { name: 'Find & filter', exact: true }).click();
  await page.locator('#treeBookmark').click();
  await expect(page.locator('#treeBookmarks button')).toHaveText(name);
});

test('passive discovery controls stay scoped to their own tab and retain the search', async ({ page }) => {
  await page.goto('/abyss/tree');
  await page.locator('#treeSearch').fill('Limit Break');
  await expect(page.locator('#treeResultCount')).toHaveText('1 result');
  for (const tab of ['#tab-talents', '#tab-specs']) {
    await page.locator(tab).click();
    await expect.soft(page.getByRole('button', { name: 'Find & filter', exact: true })).toBeHidden();
    await expect.soft(page.locator('#treeResultCount')).toBeHidden();
    await expect.soft(page.locator('#treeSearch')).toBeHidden();
    await expect.soft(page.locator('#treeFitResults')).toBeHidden();
  }
  await page.locator('#tab-passive').click();
  await expect(page.locator('#treeSearch')).toHaveValue('Limit Break');
  await expect(page.getByRole('button', { name: 'Find & filter', exact: true })).toBeVisible();
  await expect(page.locator('#treeResultCount')).toHaveText('1 result');
});

test('planning distinguishes draft activation from inspection through search and arrow keys', async ({ page }) => {
  const writes = await captureMutations(page);
  await page.goto('/abyss/tree');
  const node = await connectedNode(page);
  await page.locator('#treePlanToggle').click();
  const guidance = page.locator('#treeLegendText');
  await expect.soft(guidance).toContainText(/(?:tap|click|enter).*draft/i);
  await expect.soft(guidance).toContainText(/arrow.*inspect|inspect.*arrow/i);
  await page.locator('#treeSearch').fill(node.name);
  await page.getByRole('button', { name: 'Find & filter', exact: true }).click();
  await namedResult(page, node.name).click();
  await expect(page.locator('#treeInspectorTitle')).toHaveText(node.name);
  await expect(page.locator('#treePlanPlannedCount')).toHaveText('0');
  await page.locator('#treeSearch').fill('');
  const target = page.locator(`#treeSvg .tn[data-id="${node.id}"]`);
  await target.focus();
  await target.press('ArrowRight');
  await expect(page.locator('#treePlanPlannedCount')).toHaveText('0');
  await target.press('Enter');
  await expect(page.locator('#treePlanPlannedCount')).toHaveText('1');
  await page.locator('#treePlanToggle').click();
  await expect(guidance).toContainText(/inspect/i);
  await expect(guidance).not.toContainText(/(?:tap|click|enter).*draft/i);
  expect(writes).toEqual([]);
});

test.describe('mobile selection continuity', () => {
  test.use({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });

  test('a graph tap reveals selected effects and the explicit action without spending', async ({ page }) => {
    const writes = await captureMutations(page);
    await page.goto('/abyss/tree');
    const node = await connectedNode(page);
    await tapGraphNode(page, node.id);
    await expectSelectedDetailsInViewport(page, node.name);
    expect(writes).toEqual([]);
  });

  test('a named result reveals its inspector and returns to the same visible result', async ({ page }) => {
    const writes = await captureMutations(page);
    await page.goto('/abyss/tree');
    const node = await connectedNode(page);
    await page.locator('#treeSearch').fill(node.name);
    await page.getByRole('button', { name: 'Find & filter', exact: true }).tap();
    const result = namedResult(page, node.name);
    await result.tap();
    await expectSelectedDetailsInViewport(page, node.name);
    const back = page.locator('.tree-inspector').getByRole('button', { name: 'Back to results', exact: true });
    await expect(back).toBeInViewport({ ratio: 1 });
    await back.tap();
    await expect(result).toBeFocused();
    await expect(result).toBeInViewport({ ratio: 1 });
    await expect(page.locator('#treeSearch')).toHaveValue(node.name);
    expect(writes).toEqual([]);
  });

  test('Find & filter focuses below the sticky header and offers a visible return to the graph', async ({ page }) => {
    await page.goto('/abyss/tree');
    await page.getByRole('button', { name: 'Find & filter', exact: true }).tap();
    const sector = page.getByRole('combobox', { name: 'Sector', exact: true });
    await expect(sector).toBeFocused();
    await expect(sector).toBeInViewport({ ratio: 1 });
    const placement = await sector.evaluate(element => {
      const bounds = element.getBoundingClientRect();
      const header = document.querySelector('header');
      return { top: bounds.top, headerBottom: header ? header.getBoundingClientRect().bottom : 0 };
    });
    expect.soft(placement.top, 'Focused filters must remain below the sticky application header').toBeGreaterThanOrEqual(placement.headerBottom);
    const back = page.locator('#treeDiscoveryDetails').getByRole('button', { name: 'Back to graph', exact: true });
    await expect(back).toBeVisible();
    await back.tap();
    const entry = page.locator('#treeSvg .tn[tabindex="0"]');
    await expect(entry).toBeFocused();
    await expect(entry).toBeInViewport({ ratio: 1 });
  });

  test('a selected inspector remains usable when the layout switches between phone and desktop', async ({ page }) => {
    await page.goto('/abyss/tree');
    const node = await connectedNode(page);
    await tapGraphNode(page, node.id);
    await expectSelectedDetailsInViewport(page, node.name);
    await page.setViewportSize({ width: 1440, height: 1000 });
    await expect(page.locator('.tree-inspector')).toHaveCount(1);
    await expect(page.locator('#treeInspectorTitle')).toHaveText(node.name);
    const graph = await page.locator('#treeStage').boundingBox();
    const inspector = await page.locator('.tree-inspector').boundingBox();
    expect(inspector.x).toBeGreaterThanOrEqual(graph.x + graph.width - 1);
    await page.setViewportSize({ width: 390, height: 844 });
    await expectSelectedDetailsInViewport(page, node.name);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1)).toBe(true);
  });
});
