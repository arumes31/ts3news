(function () {
  'use strict';

  var root = document.querySelector('.ab-spectator');
  if (!root) return;

  var session = root.dataset.session || '';
  var stopped = false;
  var connection = document.getElementById('spectateConnection');

  function text(node, value) {
    if (node) node.textContent = String(value == null ? '' : value);
  }

  function finite(value) {
    var number = Number(value);
    return Number.isFinite(number) ? number : 0;
  }

  function bounded(value, minimum, maximum) {
    return Math.max(minimum, Math.min(maximum, value));
  }

  function status(kind, label) {
    if (!connection) return;
    connection.className = 'ab-spectator-connection is-' + kind;
    text(connection.querySelector('span'), label);
  }

  function unit(host, item) {
    var hp = Math.max(0, finite(item.hp));
    var maxHP = Math.max(0, finite(item.max_hp));
    var percent = maxHP > 0 ? bounded(hp * 100 / maxHP, 0, 100) : 0;
    var row = document.createElement('article');
    var summary = document.createElement('div');
    var label = document.createElement('b');
    var health = document.createElement('span');
    var meter = document.createElement('div');
    var fill = document.createElement('i');

    row.className = 'ab-spectator-unit';
    row.setAttribute('role', 'listitem');
    label.textContent = item.name || 'Unknown combatant';
    health.textContent = hp.toLocaleString() + ' / ' + maxHP.toLocaleString() + ' HP';
    meter.className = 'ab-spectator-meter';
    meter.setAttribute('role', 'meter');
    meter.setAttribute('aria-label', (item.name || 'Combatant') + ' health');
    meter.setAttribute('aria-valuemin', '0');
    meter.setAttribute('aria-valuemax', String(maxHP));
    meter.setAttribute('aria-valuenow', String(hp));
    fill.style.width = percent + '%';
    summary.append(label, health);
    meter.appendChild(fill);
    row.append(summary, meter);
    host.appendChild(row);
  }

  function renderSide(id, items) {
    var host = document.getElementById(id);
    if (!host) return;
    host.replaceChildren();
    if (!Array.isArray(items) || items.length === 0) {
      var empty = document.createElement('p');
      empty.className = 'ab-spectator-empty';
      empty.textContent = 'No combatants visible.';
      host.appendChild(empty);
      return;
    }
    items.forEach(function (item) { unit(host, item || {}); });
  }

  function renderLog(lines) {
    var host = document.getElementById('spectateLog');
    if (!host) return;
    host.replaceChildren();
    if (!Array.isArray(lines) || lines.length === 0) {
      var empty = document.createElement('p');
      empty.className = 'ab-spectator-empty';
      empty.textContent = 'No public combat events yet.';
      host.appendChild(empty);
      return;
    }
    lines.forEach(function (line) {
      var row = document.createElement('div');
      row.textContent = String(line == null ? '' : line);
      host.appendChild(row);
    });
    host.scrollTop = host.scrollHeight;
  }

  function render(state) {
    var phase = String(state.phase || 'active').toLowerCase();
    text(document.getElementById('spectateRound'), 'Round ' + Math.max(0, finite(state.round)));
    text(document.getElementById('spectatePhase'), phase.toUpperCase());
    renderSide('spectateAllies', state.allies);
    renderSide('spectateEnemies', state.enemies);
    renderLog(state.recent_logs);
    if (phase === 'complete' || phase === 'failed') {
      stopped = true;
      status('ended', phase === 'complete' ? 'Combat complete' : 'Combat ended');
      return;
    }
    status('live', 'Live now');
  }

  async function poll() {
    try {
      var response = await fetch('/api/abyss/spectate?session=' + encodeURIComponent(session), { cache: 'no-store' });
      var state = await response.json();
      if (!response.ok || !state.ok) {
        stopped = true;
        text(document.getElementById('spectatePhase'), state.error || 'Feed closed');
        status('ended', 'Feed closed');
      } else {
        render(state);
      }
    } catch (_) {
      text(document.getElementById('spectatePhase'), 'RECONNECTING');
      status('error', 'Reconnecting');
    }
    if (!stopped) window.setTimeout(poll, 900);
  }

  poll();
}());
