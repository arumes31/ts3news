const { expect } = require('@playwright/test');

function unique(values) {
  return [...new Set(values)].sort();
}

function monitorSurface(page) {
  const pageErrors = [];
  const failedAssets = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  page.on('response', response => {
    if (response.url().includes('/static/') && !response.ok()) {
      failedAssets.push(`${response.status()} ${new URL(response.url()).pathname}`);
    }
  });
  return { pageErrors, failedAssets };
}

async function expectDocumentStructure(page) {
  const issues = await page.evaluate(() => {
    const ids = new Map();
    document.querySelectorAll('[id]').forEach(node => {
      if (node.id) ids.set(node.id, (ids.get(node.id) || 0) + 1);
    });
    const duplicateIDs = [...ids]
      .filter(([, count]) => count > 1)
      .map(([id, count]) => `${id} (${count})`);
    const brokenReferences = [];
    const attributes = ['aria-activedescendant', 'aria-controls', 'aria-describedby', 'aria-labelledby', 'aria-owns'];
    document.querySelectorAll(attributes.map(name => `[${name}]`).join(',')).forEach(node => {
      attributes.forEach(name => {
        const value = node.getAttribute(name);
        if (!value) return;
        value.trim().split(/\s+/).forEach(id => {
          if (id && !document.getElementById(id)) {
            brokenReferences.push(`${node.tagName.toLowerCase()}#${node.id || '-'} ${name}=${id}`);
          }
        });
      });
    });
    return { duplicateIDs, brokenReferences };
  });
  expect(issues.duplicateIDs).toEqual([]);
  expect(issues.brokenReferences).toEqual([]);
}

async function expectVisibleSurface(page, selector, label) {
  const issues = await page.evaluate(({ selector, label }) => {
    const visible = node => {
      const style = getComputedStyle(node);
      const rect = node.getBoundingClientRect();
      return style.display !== 'none' && style.visibility !== 'hidden' && !node.hidden && rect.width > 0 && rect.height > 0;
    };
    const roots = [...document.querySelectorAll(selector)].filter(visible);
    const controls = roots
      .flatMap(root => [...root.querySelectorAll('button, input, select, textarea, a[href], [role="button"], [role="tab"]')])
      .filter(visible);
    const unnamedControls = controls.filter(node => {
      if (node.getAttribute('aria-label') || node.getAttribute('aria-labelledby') || node.getAttribute('title')) return false;
      if (node.labels && node.labels.length) return false;
      if (node.tagName === 'INPUT' && ['hidden', 'submit', 'button'].includes(node.type)) return false;
      if (['INPUT', 'SELECT', 'TEXTAREA'].includes(node.tagName)) return true;
      return !String(node.textContent || node.value || '').trim();
    }).map(node => `${label}: ${node.outerHTML.slice(0, 240)}`);
    const missingAlt = roots
      .flatMap(root => [...root.querySelectorAll('img')])
      .filter(node => visible(node) && !node.hasAttribute('alt'))
      .map(node => node.currentSrc || node.src);
    const invalidValues = [];
    roots.forEach(root => {
      const text = root.innerText || '';
      for (const match of text.matchAll(/\b(?:NaN|undefined)\b|\[object Object\]/g)) {
        const start = Math.max(0, match.index - 80);
        invalidValues.push(`${label}: ${text.slice(start, match.index + match[0].length + 80).replace(/\s+/g, ' ').trim()}`);
      }
    });
    return { rootCount: roots.length, unnamedControls, missingAlt, invalidValues };
  }, { selector, label });
  expect(issues.rootCount).toBeGreaterThan(0);
  expect(unique(issues.unnamedControls)).toEqual([]);
  expect(unique(issues.missingAlt)).toEqual([]);
  expect(unique(issues.invalidValues)).toEqual([]);
  const overflow = await page.evaluate(() => {
    const root = document.documentElement;
    const amount = root.scrollWidth - root.clientWidth;
    const offenders = [...document.querySelectorAll('body *')].filter(node => {
      const style = getComputedStyle(node);
      if (style.display === 'none' || style.visibility === 'hidden') return false;
      const rect = node.getBoundingClientRect();
      return rect.right > root.clientWidth + 1 || rect.left < -1;
    }).slice(0, 8).map(node => {
      const name = node.id ? `#${node.id}` : node.classList.length ? `.${[...node.classList].join('.')}` : node.tagName.toLowerCase();
      const rect = node.getBoundingClientRect();
      return `${name} [${rect.left.toFixed(0)}, ${rect.right.toFixed(0)}]`;
    });
    return { amount, offenders };
  });
  expect(overflow.amount, `${label} horizontal overflow: ${overflow.offenders.join(', ')}`).toBeLessThanOrEqual(1);
}

function expectSurfaceMonitorsClean(monitors) {
  expect(monitors.pageErrors).toEqual([]);
  expect(unique(monitors.failedAssets)).toEqual([]);
}

module.exports = {
  expectDocumentStructure,
  expectSurfaceMonitorsClean,
  expectVisibleSurface,
  monitorSurface,
};
