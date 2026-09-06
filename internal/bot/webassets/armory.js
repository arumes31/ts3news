(function () {
  'use strict';
  const sheet = document.querySelector('.armory-console');
  if (!sheet) return;

  const tooltip = document.createElement('div');
  tooltip.id = 'armoryItemTooltip';
  tooltip.className = 'armory-tooltip';
  tooltip.setAttribute('role', 'tooltip');
  tooltip.hidden = true;
  document.body.append(tooltip);
  let active = null;
  let closeTimer;
  let suppressed = null;
  const previewSelector = '[data-item-inspect], .gear-cell.unidentified';

  function line(parent, text, className, tag) {
    const node = document.createElement(tag || 'p');
    if (className) node.className = className;
    node.textContent = text;
    parent.append(node);
    return node;
  }
  function number(value, options) {
    return window.AbyssItemNumbers ? AbyssItemNumbers.format(value, options) : Number(value || 0).toLocaleString();
  }
  function render(trigger) {
    let item;
    if (trigger.classList.contains('unidentified')) {
      // Unidentified gear intentionally has no serialized combat record.
      item = { name: trigger.querySelector('.gear-name').textContent, slot: trigger.dataset.slot, unidentified: true };
    } else {
      try { item = JSON.parse(trigger.dataset.itemInspect); } catch (_) { return false; }
    }
    if (!item || typeof item !== 'object') return false;
    tooltip.replaceChildren();
    tooltip.style.setProperty('--tooltip-rarity', trigger.style.getPropertyValue('--gear-rarity'));
    line(tooltip, item.name, '', 'h3');
    if (!item.unidentified && !trigger.classList.contains('unidentified')) {
      line(tooltip, 'Gear score ' + number(item.score) + ' · Combat rating ' + number(item.cr, { digits: 1 }), 'tooltip-rating');
    }
    line(tooltip, [item.rarity, item.slot].filter(Boolean).join(' · '), 'tooltip-kind');
    const appearance = trigger.querySelector('.gear-appearance');
    if (appearance) line(tooltip, appearance.textContent, 'tooltip-appearance');
    if (item.unidentified || trigger.classList.contains('unidentified')) {
      line(tooltip, 'Identify this item to reveal its stats and effects. Hidden combat stats are inactive.');
    } else {
      if (Array.isArray(item.stats) && item.stats.length) {
        const stats = line(tooltip, '', 'tooltip-stats', 'div');
        item.stats.forEach(stat => line(stats, number(stat.value, { sign: true }) + ' ' + stat.label));
      }
      if (Array.isArray(item.specials) && item.specials.length) {
        const effects = line(tooltip, '', 'tooltip-effects', 'div');
        item.specials.forEach(effect => line(effects, effect.name + (effect.description ? ': ' + effect.description : '')));
      }
      if (item.element) line(tooltip, 'Element: ' + item.element);
      if (item.set_id) line(tooltip, 'Set: ' + item.set_id);
      if (item.xp_bonus_pct) line(tooltip, number(item.xp_bonus_pct, { sign: true }) + '% XP');
      if (item.max_durability) {
        const durability = Number(trigger.dataset.itemDurability);
        line(tooltip, (durability === 0 ? 'Broken · ' : '') + durability + ' / ' + item.max_durability + ' durability');
      }
      if (trigger.dataset.itemBrokenIn === 'true') line(tooltip, 'Broken in · +1% stats');
      if (Array.isArray(item.modifiers)) item.modifiers.forEach(modifier => line(tooltip, modifier));
      if (Array.isArray(item.gemstones)) item.gemstones.forEach(gem => line(tooltip, 'Gem: ' + gem));
      if (item.provenance) line(tooltip, item.provenance, 'tooltip-provenance');
    }
    line(tooltip, item.unidentified ? 'Use Identify item to review the cost · Esc to dismiss' : 'Select for the complete record · Esc to dismiss', 'tooltip-hint');
    return true;
  }
  function position() {
    if (!active || tooltip.hidden) return;
    const anchor = active.getBoundingClientRect();
    const width = tooltip.offsetWidth;
    const height = tooltip.offsetHeight;
    const viewportWidth = document.documentElement.clientWidth;
    const viewportHeight = window.innerHeight;
    let left = anchor.right + 10;
    if (left + width > viewportWidth - 8) left = anchor.left - width - 10;
    left = Math.max(8, Math.min(left, viewportWidth - width - 8));
    const top = Math.max(8, Math.min(anchor.top, viewportHeight - height - 8));
    tooltip.style.left = left + 'px';
    tooltip.style.top = top + 'px';
  }
  function hide() {
    clearTimeout(closeTimer);
    if (active) {
      const descriptions = (active.getAttribute('aria-describedby') || '').split(' ').filter(id => id && id !== tooltip.id);
      if (descriptions.length) active.setAttribute('aria-describedby', descriptions.join(' '));
      else active.removeAttribute('aria-describedby');
    }
    tooltip.hidden = true;
    active = null;
  }
  function show(trigger) {
    clearTimeout(closeTimer);
    if (!trigger || trigger === suppressed || document.querySelector('#sharedModal.open')) return;
    if (active === trigger) return;
    hide();
    if (!render(trigger)) return;
    active = trigger;
    tooltip.hidden = false;
    const descriptions = (trigger.getAttribute('aria-describedby') || '').split(' ').filter(Boolean);
    trigger.setAttribute('aria-describedby', [...descriptions, tooltip.id].join(' '));
    position();
  }
  function scheduleHide() {
    clearTimeout(closeTimer);
    closeTimer = setTimeout(hide, 160);
  }
  sheet.addEventListener('pointerover', function (event) {
    if (event.pointerType === 'touch') return;
    const trigger = event.target.closest(previewSelector);
    if (trigger && !trigger.contains(event.relatedTarget)) {
      suppressed = null;
      show(trigger);
    }
  });
  sheet.addEventListener('pointerout', function (event) {
    const trigger = event.target.closest(previewSelector);
    if (trigger && !trigger.contains(event.relatedTarget)) scheduleHide();
  });
  sheet.addEventListener('focusin', function (event) {
    suppressed = null;
    show(event.target.closest(previewSelector));
  });
  sheet.addEventListener('focusout', function (event) {
    if (active && !active.contains(event.relatedTarget)) scheduleHide();
  });
  tooltip.addEventListener('pointerenter', function () { clearTimeout(closeTimer); });
  tooltip.addEventListener('pointerleave', scheduleHide);
  // Suppress the preview before the shared click/keyboard inspector opens.
  sheet.addEventListener('click', function () { suppressed = active; hide(); }, true);
  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' || event.key === 'Enter' || event.key === ' ') {
      suppressed = active;
      hide();
    }
  }, true);
  window.addEventListener('scroll', function () {
    if (!active) return;
    const rect = active.getBoundingClientRect();
    if (rect.bottom < 0 || rect.top > window.innerHeight) hide();
    else position();
  }, { passive: true });
  window.addEventListener('resize', position);
  document.addEventListener('abyssitemnumberschange', function () {
    if (active) { render(active); position(); }
  });
  sheet.querySelectorAll('.armory-sections a').forEach(link => {
    link.addEventListener('click', function () {
      sheet.querySelectorAll('.armory-sections a').forEach(other => other.removeAttribute('aria-current'));
      link.setAttribute('aria-current', 'location');
    });
  });
})();
