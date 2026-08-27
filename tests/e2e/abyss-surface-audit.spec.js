const { test, expect } = require('@playwright/test');

const sections = ['season', 'progression', 'observatory', 'shop', 'forge', 'social', 'lore', 'leaderboards'];
const viewports = [
  { name: 'desktop', width: 1440, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
];

function unique(values) {
  return [...new Set(values)].sort();
}

for (const viewport of viewports) {
  test(`every Abyss workspace has a valid rendered surface on ${viewport.name}`, async ({ page }) => {
    const pageErrors = [];
    const failedAssets = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    page.on('response', response => {
      if (response.url().includes('/static/') && !response.ok()) {
        failedAssets.push(`${response.status()} ${new URL(response.url()).pathname}`);
      }
    });
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.emulateMedia({ reducedMotion: 'reduce', colorScheme: 'light' });
    await page.goto('/abyss?gear=1');

    const documentIssues = await page.evaluate(() => {
      const ids = new Map();
      document.querySelectorAll('[id]').forEach(node => {
        const id = node.id;
        if (id) ids.set(id, (ids.get(id) || 0) + 1);
      });
      const duplicateIDs = [...ids].filter(([, count]) => count > 1).map(([id, count]) => `${id} (${count})`);
      const brokenReferences = [];
      const referenceAttributes = ['aria-activedescendant', 'aria-controls', 'aria-describedby', 'aria-labelledby', 'aria-owns'];
      document.querySelectorAll(referenceAttributes.map(name => `[${name}]`).join(',')).forEach(node => {
        referenceAttributes.forEach(name => {
          const value = node.getAttribute(name);
          if (!value) return;
          value.trim().split(/\s+/).forEach(id => {
            if (id && !document.getElementById(id)) brokenReferences.push(`${node.tagName.toLowerCase()}#${node.id || '-'} ${name}=${id}`);
          });
        });
      });
      return { duplicateIDs, brokenReferences };
    });
    expect(documentIssues.duplicateIDs).toEqual([]);
    expect(documentIssues.brokenReferences).toEqual([]);

    for (const key of sections) {
      const tab = page.locator(`[data-tab-key="${key}"]`);
      await tab.click();
      await expect(tab).toHaveAttribute('aria-selected', 'true');
      const issues = await page.evaluate(activeKey => {
        const visible = node => {
          const style = getComputedStyle(node);
          const rect = node.getBoundingClientRect();
          return style.display !== 'none' && style.visibility !== 'hidden' && !node.hidden && rect.width > 0 && rect.height > 0;
        };
        const panels = [...document.querySelectorAll(`[data-abyss-section="${activeKey}"]`)].filter(visible);
        const controls = panels.flatMap(panel => [...panel.querySelectorAll('button, input, select, textarea, a[href], [role="button"], [role="tab"]')]).filter(visible);
        const unnamedControls = controls.filter(node => {
          if (node.getAttribute('aria-label') || node.getAttribute('aria-labelledby') || node.getAttribute('title')) return false;
          if (node.labels && node.labels.length) return false;
          if (node.tagName === 'INPUT' && ['hidden', 'submit', 'button'].includes(node.type)) return false;
          if (['INPUT', 'SELECT', 'TEXTAREA'].includes(node.tagName)) return true;
          return !String(node.textContent || node.value || '').trim();
        }).map(node => `${activeKey}: ${node.outerHTML.slice(0, 240)}`);
        const missingAlt = panels.flatMap(panel => [...panel.querySelectorAll('img')]).filter(node => visible(node) && !node.hasAttribute('alt')).map(node => node.currentSrc || node.src);
        const invalidValues = [];
        panels.forEach(panel => {
          const text = panel.innerText;
          for (const match of text.matchAll(/(^|\s)(NaN|undefined|\[object Object\])(?=\s|$)/g)) {
            const start = Math.max(0, match.index - 80);
            invalidValues.push(`${activeKey}: ${text.slice(start, match.index + match[0].length + 80).replace(/\s+/g, ' ').trim()}`);
          }
        });
        return { panelCount: panels.length, unnamedControls, missingAlt, invalidValues };
      }, key);
      expect(issues.panelCount).toBeGreaterThan(0);
      expect(unique(issues.unnamedControls)).toEqual([]);
      expect(unique(issues.missingAlt)).toEqual([]);
      expect(unique(issues.invalidValues)).toEqual([]);
      expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(1);
    }

    expect(pageErrors).toEqual([]);
    expect(unique(failedAssets)).toEqual([]);
  });
}
