const { test, expect } = require('@playwright/test');

test.describe('Restrained live region filter announcements (Proposal 0141)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('live regions have role="status", aria-live="polite", and aria-atomic="true"', async ({ page }) => {
    const campaignCount = page.locator('#rift-filter-count');
    await expect(campaignCount).toHaveAttribute('role', 'status');
    await expect(campaignCount).toHaveAttribute('aria-live', 'polite');
    await expect(campaignCount).toHaveAttribute('aria-atomic', 'true');

    const monsterMatches = page.locator('#rift-monster-matches');
    await expect(monsterMatches).toHaveAttribute('role', 'status');
    await expect(monsterMatches).toHaveAttribute('aria-live', 'polite');
    await expect(monsterMatches).toHaveAttribute('aria-atomic', 'true');
  });

  test('campaign filter live region debounces typing but updates immediately on select and clear', async ({ page }) => {
    const campaignCount = page.locator('#rift-filter-count');
    await expect(campaignCount).toHaveText('100 of 100 missions');
    await expect(campaignCount).toHaveAttribute('aria-label', 'Showing all 100 missions.');

    const searchInput = page.locator('#rift-mission-search');

    // Type in search: visual cards update immediately, but live region is debounced
    await searchInput.pressSequentially('10', { delay: 10 });

    // After debounce timer (250ms), live region reflects search matches
    await expect(campaignCount).not.toHaveText('100 of 100 missions');
    await expect(campaignCount).toHaveAttribute('aria-label', /missions matching filters/);

    // Click "Clear filters": updates immediately
    const clearBtn = page.locator('#rift-clear-filters');
    await clearBtn.click();
    expect(await campaignCount.textContent()).toBe('100 of 100 missions');
    expect(await campaignCount.getAttribute('aria-label')).toBe('Showing all 100 missions.');

    // Select dropdown: updates immediately
    const diffSelect = page.locator('#rift-difficulty');
    await diffSelect.selectOption('Mythic');
    const mythicCount = await campaignCount.textContent();
    expect(mythicCount).not.toBe('100 of 100 missions');
    expect(mythicCount).toMatch(/\d+ of 100 missions/);
    expect(await campaignCount.getAttribute('aria-label')).toMatch(/\d+ of 100 missions matching filters\./);

    // When 0 missions match
    await clearBtn.click();
    await searchInput.fill('xyznonexistentquery999');
    await page.waitForTimeout(250);
    await expect(campaignCount).toHaveText('0 of 100 missions');
    await expect(campaignCount).toHaveAttribute('aria-label', 'No missions match current filters. 0 of 100 missions.');
  });

  test('bestiary filter live region debounces typing and updates immediately on controls', async ({ page }) => {
    // Open bestiary details
    await page.locator('details.rift-bestiary > summary').click();

    const monsterMatches = page.locator('#rift-monster-matches');
    await expect(monsterMatches).toHaveText(/\d+ of \d+ monsters/);
    const totalText = await monsterMatches.textContent();
    const totalMatch = totalText.match(/^(\d+) of (\d+) monsters$/);
    expect(totalMatch).toBeTruthy();
    const totalCount = totalMatch[1];
    await expect(monsterMatches).toHaveAttribute('aria-label', `Showing all ${totalCount} monsters.`);

    // Type in monster search
    const monsterSearch = page.locator('#rift-monster-search');
    await monsterSearch.pressSequentially('boss', { delay: 10 });

    // After debounce (250ms), matches update
    await expect(monsterMatches).not.toHaveText(`${totalCount} of ${totalCount} monsters`);
    await expect(monsterMatches).toHaveAttribute('aria-label', /matching filters/);

    // Clear filters button: immediate update
    const clearBtn = page.locator('#rift-monster-clear');
    await clearBtn.click();
    expect(await monsterMatches.textContent()).toBe(`${totalCount} of ${totalCount} monsters`);
    expect(await monsterMatches.getAttribute('aria-label')).toBe(`Showing all ${totalCount} monsters.`);

    // Dropdown change: immediate update
    const tierSelect = page.locator('#rift-monster-tier');
    const firstTierOption = await tierSelect.locator('option').nth(1).getAttribute('value');
    if (firstTierOption) {
      await tierSelect.selectOption(firstTierOption);
      const tierFilteredText = await monsterMatches.textContent();
      expect(tierFilteredText).not.toBe(`${totalCount} of ${totalCount} monsters`);
      expect(await monsterMatches.getAttribute('aria-label')).toMatch(/matching filters\./);
    }
  });

  test('suppresses redundant DOM mutations when counts do not change', async ({ page }) => {
    // Setup mutation observer on #rift-filter-count
    await page.evaluate(() => {
      window.__filterMutations = 0;
      const target = document.getElementById('rift-filter-count');
      const observer = new MutationObserver(() => {
        window.__filterMutations++;
      });
      observer.observe(target, { characterData: true, childList: true, attributes: true, subtree: true });
    });

    // Calling RiftCampaignTools.update without changing filters or counts
    await page.evaluate(() => {
      window.RiftCampaignTools.update(null, 1);
      window.RiftCampaignTools.update(null, 1);
    });

    const mutationCount = await page.evaluate(() => window.__filterMutations);
    expect(mutationCount).toBe(0);
  });
});
