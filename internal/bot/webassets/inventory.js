/* Inventory mutations settle against server state before another purchase. */
var inventoryPending = false;
var inventoryNeedsRefresh = false;

function toast(message, ok) {
  var status = document.getElementById('invMsg');
  status.textContent = message;
  status.className = 'inventory-status ' + (ok ? 'good' : 'bad');
}

function inspectInventoryItem(button) {
  openItemInspector(button);
  var inspector = document.querySelector('#sharedModal .item-inspector');
  if (inspector) inspector.style.setProperty('--inspect-rarity', getComputedStyle(button).color);
}

function inventoryGold(value) {
  var gold = document.getElementById('goldPill');
  if (!gold) return;
  gold.dataset.value = String(value);
  gold.textContent = '🪙 ' + Number(value || 0).toLocaleString();
}

function inventoryFilter() {
  var select = document.getElementById('inventorySlot');
  var cards = Array.from(document.querySelectorAll('.inv-card[data-slot]'));
  var requested = new URLSearchParams(location.search).get('slot') || '';
  if (requested.length > 40) requested = '';
  var slots = Array.from(new Set(cards.map(function(card) { return card.dataset.slot; })));
  if (requested && slots.indexOf(requested) < 0) slots.push(requested);
  slots.sort().forEach(function(slot) {
    var option = document.createElement('option');
    option.value = slot; option.textContent = slot; select.appendChild(option);
  });
  select.value = requested;
  function filter() {
    var shown = 0;
    cards.forEach(function(card) { card.hidden = !!select.value && card.dataset.slot !== select.value; if (!card.hidden) shown++; });
    document.getElementById('inventorySlotStatus').textContent = shown + ' item' + (shown === 1 ? '' : 's') + (select.value ? ' for ' + select.value : ' across all slots');
    document.getElementById('inventorySlotEmpty').hidden = !select.value || shown > 0;
  }
  select.addEventListener('change', function() {
    filter();
    var url = new URL(location.href);
    if (select.value) url.searchParams.set('slot', select.value); else url.searchParams.delete('slot');
    history.replaceState(null, '', url.pathname + url.search + '#inventoryGear');
  });
  filter();
}

async function refreshInventory(all) {
  var response = await fetch(location.pathname + location.search, { cache: 'no-store', signal: AbortSignal.timeout(15000) });
  if (!response.ok || response.redirected) throw new Error('Inventory unavailable');
  var doc = new DOMParser().parseFromString(await response.text(), 'text/html');
  var ids = all ? ['inventoryItems', 'vendorBuybacks', 'pouchTailoring', 'goldPill'] : ['inventoryItems'];
  if (ids.some(function(id) { return !doc.getElementById(id); })) throw new Error('Inventory unavailable');
  ids.forEach(function(id) { document.getElementById(id).replaceWith(doc.getElementById(id)); });
  inventoryFilter();
  if (window.AbyssItemNumbers) AbyssItemNumbers.render(document.getElementById('inventoryItems'));
}

function inventoryBusy(button, busy) {
  if (!button) return;
  button.disabled = busy;
  if (busy) button.setAttribute('aria-busy', 'true'); else button.removeAttribute('aria-busy');
}

async function inventoryAction(button, url, body, success, confirmation) {
  if (inventoryPending || inventoryNeedsRefresh || !button || button.disabled) return;
  inventoryPending = true;
  inventoryBusy(button, true);
  var confirmed = false;
  try {
    if (confirmation && !await confirmation()) return;
    toast('Completing action…', true);
    var response = await fetch(url, {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body), signal: AbortSignal.timeout(15000)
    });
    var data = await response.json();
    if (!response.ok || !data || typeof data.ok !== 'boolean') throw new Error('Unconfirmed response');
    if (!data.ok) { toast(data.error || 'The action could not be completed.', false); return; }
    confirmed = true;
    await success(data);
  } catch (_) {
    // A transport failure may happen after the server committed the purchase.
    // Re-read all balances and items before enabling another action.
    try {
      await refreshInventory(true);
      toast(confirmed ? 'Action completed. Inventory refreshed.' : 'Could not confirm the action. Inventory refreshed; check your items and balance before trying again.', confirmed);
      document.getElementById('inventorySlot').focus({ preventScroll: true });
    } catch (_) {
      inventoryNeedsRefresh = true;
      document.getElementById('inventoryRefresh').hidden = false;
      document.querySelectorAll('.inv-actions button, .buyback-card button, #pouchUpgradeButton').forEach(function(action) { action.disabled = true; });
      toast(confirmed ? 'Action completed, but inventory could not refresh. Refresh inventory to see your changes.' : 'Could not confirm the action or refresh inventory. Reconnect and refresh before trying again.', false);
      document.getElementById('inventoryRefresh').focus({ preventScroll: true });
    }
  } finally {
    inventoryPending = false;
    inventoryBusy(button, inventoryNeedsRefresh || button.dataset.masterwork === 'true');
    if (!confirmed) window.setTimeout(function() {
      if (button.isConnected && !button.disabled && !document.getElementById('sharedModal').classList.contains('open')) button.focus({ preventScroll: true });
    }, 0);
  }
}

function equip(id, button) {
  return inventoryAction(button, '/api/inventory/equip', { inv_id: id }, async function(data) {
    await refreshInventory(true);
    toast('Equipped ' + data.equipped, true);
    document.getElementById('inventorySlot').focus({ preventScroll: true });
  });
}

function sell(id, button) {
  return inventoryAction(button, '/api/inventory/sell', { inv_id: id }, async function(data) {
    await refreshInventory(true);
    toast('Vendored for ' + Number(data.value).toLocaleString() + ' gold.', true);
    document.getElementById('inventorySlot').focus({ preventScroll: true });
  });
}

function buyback(button) {
  var cost = Number(button.dataset.cost || 0);
  return inventoryAction(button, '/api/inventory/buyback', { id: Number(button.dataset.id) }, async function(data) {
    button.closest('.buyback-card').remove();
    var list = document.getElementById('vendorBuybackList');
    if (!list.querySelector('.buyback-card')) {
      var empty = document.createElement('p');
      empty.className = 'buyback-empty';
      empty.textContent = 'No recent vendor sales. Your next ten eligible sales will appear here.';
      list.appendChild(empty);
    }
    inventoryGold(data.gold);
    await refreshInventory(false);
    toast(data.msg || 'Item returned to your inventory.', true);
    document.getElementById('inventorySlot').focus({ preventScroll: true });
  }, function() {
    return confirmModal('Restore this exact item for 🪙 ' + cost.toLocaleString() + '?<br><span class="muted small">Includes the disclosed 10% vendor handling fee.</span>', { title: 'Confirm vendor buyback', okLabel: 'Buy back · ' + cost.toLocaleString() + 'g' });
  });
}

function upgradePouch(button) {
  var cost = Number(button.dataset.cost || 0);
  return inventoryAction(button, '/api/inventory/pouch/upgrade', {}, function(data) {
    document.getElementById('pouchRank').textContent = String(data.level);
    document.getElementById('pouchStackCap').textContent = String(data.stack_cap);
    document.getElementById('pouchCarryCap').textContent = String(data.carry_cap);
    document.querySelectorAll('.pouch-ranks i').forEach(function(rank, index) { rank.classList.toggle('filled', index < data.level); });
    var next = [Number(data.stack_cap) + 1, Number(data.carry_cap) + 1];
    document.querySelectorAll('.pouch-caps small').forEach(function(label, index) { label.hidden = !(data.next_cost > 0); label.textContent = 'next ' + next[index]; });
    if (data.next_cost > 0) {
      button.dataset.cost = String(data.next_cost);
      button.textContent = 'Tailor rank ' + (data.level + 1) + ' · 🪙 ' + Number(data.next_cost).toLocaleString();
    } else {
      button.dataset.masterwork = 'true';
      button.textContent = '✓ Masterwork';
      document.querySelector('.pouch-copy p').textContent = 'Masterwork tailoring complete.';
      document.getElementById('pouchTitle').setAttribute('tabindex', '-1');
      document.getElementById('pouchTitle').focus({ preventScroll: true });
    }
    inventoryGold(data.gold);
    toast(data.msg || 'Consumable Pouch tailored.', true);
  }, function() {
    return confirmModal('Permanently raise both pouch limits for 🪙 ' + cost.toLocaleString() + '?<br><span class="muted small">+1 Abyss loot stack · +1 equipped run carry.</span>', { title: 'Tailor Consumable Pouch', okLabel: 'Tailor · ' + cost.toLocaleString() + 'g' });
  });
}

document.querySelector('.inventory-page').addEventListener('click', function(event) {
  if (event.target.closest('button,a,input,select,textarea')) return;
  var card = event.target.closest('.inv-card');
  var button = card && card.querySelector('.inv-inspect');
  if (button) { button.focus({ preventScroll: true }); inspectInventoryItem(button); }
});
inventoryFilter();
