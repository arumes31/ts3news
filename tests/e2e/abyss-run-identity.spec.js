const { test, expect } = require('@playwright/test');

const choices = [
  { id: 1, key: 'cinder', name: 'Cinder March', icon: '🔥', affinity: 'fire', promise: '+volatile cache pressure', warning: 'hotter, harder floors' },
  { id: 2, key: 'verdant', name: 'Verdant Coil', icon: '🌿', affinity: 'nature', promise: '+steady mastery routes', warning: 'rootbound attrition' },
  { id: 3, key: 'tempest', name: 'Tempest Crown', icon: '⚡', affinity: 'storm', promise: '+fast elite encounters', warning: 'storm-wracked danger' },
  { id: 4, key: 'void', name: 'Void Pilgrimage', icon: '◉', affinity: 'void', promise: '+deep-biome mastery', warning: 'the deadliest route' },
];

function identity(overrides = {}) {
  return {
    active: true,
    story: true,
    story_complete: false,
    story_progress: 6,
    story_beats: [],
    biome_until: 0,
    biome_choice_required: false,
    biome_choices: choices,
    relics: [{ id: 1, key: 'ember_lens', name: 'Ember Lens', icon: '◈', effect: '+12% STR this run' }],
    boons: [{ id: 2, key: 'iron_oath', name: 'Iron Oath', icon: '⬡', effect: '+8% DEF per stack', stacks: 2 }],
    draft: { pending: false, depth: 0, options: [] },
    next_relic_depth: 8,
    ...overrides,
  };
}

for (const viewport of [
  { name: 'desktop', width: 1440, height: 1000 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  test(`expedition chronicle resolves drafts and biome contracts on ${viewport.name}`, async ({ page }) => {
    const pageErrors = [];
    const requests = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    await page.setViewportSize(viewport);
    await page.route('**/api/abyss/run/**', async route => {
      const path = new URL(route.request().url()).pathname;
      const body = route.request().postDataJSON() || {};
      requests.push({ path, body });
      if (path.endsWith('/boon')) {
        await route.fulfill({
          contentType: 'application/json',
          body: JSON.stringify({ ok: true, name: "Giant's Favor", stacks: 1, run_identity: identity({
            boons: [{ id: 1, key: 'giants_favor', name: "Giant's Favor", icon: '⚔', effect: '+8% STR per stack', stacks: 1 }],
          }) }),
        });
        return;
      }
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ ok: true, name: 'Cinder March', until: 11, run_identity: identity({ biome: choices[0], biome_until: 11 }) }),
      });
    });

    await page.goto('/abyss?active=1&chronicle=draft');
    const chronicle = page.locator('#abyssRunIdentity');
    await expect(chronicle).toBeVisible();
    await expect(chronicle.locator('.ab-story-track li')).toHaveCount(10);
    await expect(chronicle.locator('#abyssBoonCard')).toHaveClass(/needs-choice/);
    await expect(chronicle.locator('.ab-boon-draft button')).toHaveCount(3);
    await chronicle.locator('.ab-boon-draft button').first().click();
    await expect(chronicle.locator('#abyssRunBoons')).toContainText("Giant's Favor ×1");
    await expect(chronicle.locator('.ab-boon-draft')).toHaveCount(0);

    await page.goto('/abyss?active=1&chronicle=rest');
    await expect(page.locator('#nonCombatPanel')).toBeVisible();
    const restChoices = page.locator('#ncOptions .ab-biome-choice-list');
    await expect(restChoices.getByRole('button')).toHaveCount(4);
    await restChoices.getByRole('button', { name: /Cinder March/ }).click();
    await expect(page.locator('#abyssBiomeCard')).toContainText('Cinder March');
    await expect(page.locator('#ncOptions .ab-biome-choice-list')).toHaveCount(0);

    await expect.poll(() => requests.some(request => request.path.endsWith('/boon') && request.body.id === 1)).toBe(true);
    await expect.poll(() => requests.some(request => request.path.endsWith('/biome') && request.body.id === 1)).toBe(true);
    expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    expect(pageErrors).toEqual([]);
  });
}
