const { test, expect } = require('@playwright/test');

for (const viewport of [
  { name: 'desktop', width: 1440, height: 1000 },
  { name: 'mobile', width: 390, height: 844 },
]) {
  test(`shared accessibility and QoL controls remain coherent on ${viewport.name}`, async ({ page }) => {
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    await page.setViewportSize(viewport);
    await page.addInitScript(() => {
      localStorage.setItem('ab_contrast', '1');
      localStorage.setItem('ab_colorblind', '1');
      localStorage.setItem('ab_player_cv', 'protan');
      localStorage.setItem('ab_player_motion', 'reduce');
      localStorage.setItem('ab_logverbosity', 'summary');
      localStorage.setItem('ab_logmono', '1');
    });
    await page.goto('/abyss?active=1');

    await expect(page.locator('body')).toHaveClass(/ab-high-contrast/);
    await expect(page.locator('body')).toHaveClass(/ab-colorblind/);
    await expect(page.locator('body')).toHaveClass(/ab-cv-protan/);
    await expect(page.locator('body')).toHaveClass(/ab-user-reduce-motion/);
    await expect(page.locator('html')).toHaveAttribute('data-ab-log-verbosity', 'summary');
    await expect(page.locator('#abyssLog')).toHaveClass(/ab-mono/);

    const audit = await page.evaluate(() => {
      const scope = document.createElement('section');
      scope.id = 'accessAuditFixture';
      scope.innerHTML = '<h2>🔔 Updates</h2><output data-emoji-icon>⚠️</output><button></button>';
      document.querySelector('.abyss-command-page').appendChild(scope);
      window.__abyssAccessibility.enhanceStaticEmoji(scope);
      const before = window.__abyssAccessibility.audit(scope);
      scope.querySelector('button').setAttribute('aria-label', 'Continue safely');
      const after = window.__abyssAccessibility.audit(scope);
      return { before, after };
    });
    expect(audit.before).toEqual([{ kind: 'missing-name', id: 'button' }]);
    expect(audit.after).toEqual([]);
    await expect(page.locator('#accessAuditFixture h2 .ab-emoji-icon')).toHaveAttribute('aria-hidden', 'true');
    await expect(page.locator('#accessAuditFixture output')).toHaveAttribute('aria-label', 'Warning');

    await page.keyboard.press('Tab');
    await expect(page.locator('.ab-skip-link')).toBeFocused();
    if (viewport.name === 'mobile') {
      await expect(page.locator('#abyssMobileActions button').first()).toHaveCSS('min-height', '48px');
    }
    expect(pageErrors).toEqual([]);
  });
}
