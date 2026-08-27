(function () {
  'use strict';
  const $ = id => document.getElementById(id);
  const root = $('abyssOps');
  if (!root) return;
  let token = sessionStorage.getItem('abyss_ops_token') || '';
  let loading = false;

  function setStatus(message, kind) {
    const status = $('opsStatus');
    status.textContent = message;
    status.className = 'ops-status' + (kind ? ' ' + kind : '');
  }

  async function api(method, body) {
    if (!token) throw new Error('Enter the dedicated operator token.');
    const response = await fetch('/api/abyss/ops', {
      method,
      cache: 'no-store',
      headers: { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' },
      body: body ? JSON.stringify(body) : undefined,
    });
    if (response.status === 404) throw new Error('Operator access was not accepted.');
    const payload = await response.json().catch(() => ({}));
    if (!response.ok || payload.ok === false) throw new Error(payload.error || 'Operator request failed.');
    return payload;
  }

  function pct(value, digits) { return (Number(value || 0) * 100).toFixed(digits) + '%'; }
  function number(value, digits) { return Number(value || 0).toLocaleString(undefined, { maximumFractionDigits: digits }); }

  function renderChart(id, days, key, format, emptyID) {
    const svg = $(id);
    svg.replaceChildren();
    svg.classList.toggle('drop', key === 'drops_per_floor');
    const values = days.map(day => Number(day[key] || 0));
    $(emptyID).hidden = values.length > 0;
    if (!values.length) return;
    const ns = 'http://www.w3.org/2000/svg', width = 720, height = 250, pad = 34;
    svg.setAttribute('viewBox', `0 0 ${width} ${height}`);
    const maximum = Math.max(...values, key === 'death_rate' ? 0.01 : 0.1);
    const x = index => pad + index * (width - pad * 2) / Math.max(1, values.length - 1);
    const y = value => height - pad - value / maximum * (height - pad * 2);
    for (let line = 0; line <= 4; line++) {
      const yy = pad + line * (height - pad * 2) / 4;
      const grid = document.createElementNS(ns, 'line');
      grid.setAttribute('class', 'grid'); grid.setAttribute('x1', pad); grid.setAttribute('x2', width - pad); grid.setAttribute('y1', yy); grid.setAttribute('y2', yy); svg.appendChild(grid);
      const label = document.createElementNS(ns, 'text');
      label.setAttribute('x', 0); label.setAttribute('y', yy + 4); label.textContent = format(maximum * (4 - line) / 4); svg.appendChild(label);
    }
    const path = document.createElementNS(ns, 'path');
    path.setAttribute('class', 'line');
    path.setAttribute('d', values.map((value, index) => `${index ? 'L' : 'M'}${x(index)} ${y(value)}`).join(' '));
    svg.appendChild(path);
    values.forEach((value, index) => {
      const point = document.createElementNS(ns, 'circle');
      point.setAttribute('cx', x(index)); point.setAttribute('cy', y(value)); point.setAttribute('r', 3.5);
      const title = document.createElementNS(ns, 'title'); title.textContent = `${days[index].date}: ${format(value)}`; point.appendChild(title); svg.appendChild(point);
    });
    const first = document.createElementNS(ns, 'text'); first.setAttribute('x', pad); first.setAttribute('y', height - 5); first.textContent = days[0].date.slice(5); svg.appendChild(first);
    const last = document.createElementNS(ns, 'text'); last.setAttribute('x', width - pad); last.setAttribute('y', height - 5); last.setAttribute('text-anchor', 'end'); last.textContent = days.at(-1).date.slice(5); svg.appendChild(last);
  }

  function renderFeatures(features) {
    root.querySelectorAll('[data-ops-feature]').forEach(input => { input.checked = Boolean(features[input.dataset.opsFeature]); input.disabled = false; });
    const rollout = (id, value) => { const percent = Number(value ?? features.rollout_percent ?? 0); $(id).value = percent; $(id + 'Value').textContent = percent + '%'; };
    rollout('liveRollout', features.rollout_percent);
    rollout('socialRollout', features.social_rollout_percent);
    rollout('treeRollout', features.tree_rollout_percent);
    rollout('forgeRollout', features.forge_rollout_percent);
    $('rewardRollout').value = features.reward_experiment_rollout_percent; $('rewardRolloutValue').textContent = features.reward_experiment_rollout_percent + '%';
    $('rewardBonus').value = features.reward_treatment_bonus_bps; $('rewardBonusValue').textContent = '+' + (features.reward_treatment_bonus_bps / 100).toFixed(2) + '%';
    $('opsRevision').textContent = 'revision ' + features.revision;
  }

  function renderFunnel(funnel) {
    const stops = funnel.stops_by_depth || {}, order = ['entry', '1-4', '5-9', '10-24', '25-49', '50-99', '100+'];
    const bands = order.filter(band => stops[band]);
    Object.keys(stops).sort().forEach(band => { if (!bands.includes(band)) bands.push(band); });
    const rows = $('opsFunnelRows'); rows.replaceChildren();
    if (!bands.length) { const row = rows.insertRow(); const cell = row.insertCell(); cell.colSpan = 6; cell.textContent = 'No completed runs sampled.'; }
    bands.forEach(band => {
      const reasons = stops[band] || {}, row = rows.insertRow();
      [band, reasons.banked, reasons.conceded, reasons.timeout, reasons.revive_failed, reasons.other].forEach((value, index) => {
        const cell = row.insertCell(); cell.textContent = index ? number(value, 0) : value;
      });
    });
    $('opsFunnelScope').textContent = String(funnel.scope || 'process lifetime').replace('_', ' ');
    $('opsFunnelSummary').textContent = `${number(funnel.entered, 0)} entered · ${number(funnel.reached_floor_5, 0)} reached floor 5 · ${number(funnel.banked, 0)} banked · ${number(funnel.conceded, 0)} forfeited · ${number(funnel.active_tracked, 0)} active`;
  }

  function renderExperiment(experiment) {
    const status = $('opsExperimentStatus');
    status.textContent = String(experiment.status || 'disabled').replace('_', ' ');
    status.className = 'ops-badge ' + (experiment.status || 'disabled');
    const rows = $('opsCohorts'); rows.replaceChildren();
    const cohorts = experiment.cohorts || {}, names = ['control', 'treatment', 'holdout'].filter(name => cohorts[name]);
    if (!names.length) { const row = rows.insertRow(); const cell = row.insertCell(); cell.colSpan = 5; cell.textContent = 'No experiment samples.'; return; }
    names.forEach(name => {
      const data = cohorts[name], row = rows.insertRow();
      [name, number(data.floors, 0), number(data.average_reward, 1), pct(data.death_rate, 1), pct(data.anomaly_rate, 1)].forEach(value => { const cell = row.insertCell(); cell.textContent = value; });
    });
  }

  function render(payload) {
    const registry = payload.registry || {}, latency = payload.latency_ms || {}, actions = payload.actions || {}, anomalies = payload.anomalies || {};
    $('opsActive').textContent = number(registry.active, 0); $('opsRegistry').textContent = `${number(registry.stale, 0)} stale · ${number(registry.orphan, 0)} orphan`;
    $('opsLatency').textContent = `${number(latency.request_avg, 0)} / ${number(latency.request_max, 0)} ms`;
    $('opsAutomation').textContent = pct(actions.automatic_rate, 1); $('opsActions').textContent = `${number(actions.automatic, 0)} auto · ${number(actions.manual, 0)} manual`;
    $('opsAnomalies').textContent = number(anomalies.total, 0); $('opsAnomalySplit').textContent = `${number(anomalies.reward, 0)} reward · ${number(anomalies.damage, 0)} damage · ${number(anomalies.economy, 0)} economy`;
    renderFeatures(payload.features || {}); renderExperiment(payload.reward_experiment || {}); renderFunnel(payload.funnel || {});
    const balance = payload.balance || {}, days = balance.available ? (balance.days || []) : [];
    renderChart('opsDeathChart', days, 'death_rate', value => pct(value, 0), 'opsDeathEmpty');
    renderChart('opsDropChart', days, 'drops_per_floor', value => number(value, 2), 'opsDropEmpty');
    if (!balance.available) { $('opsDeathEmpty').textContent = 'Historical balance data is unavailable.'; $('opsDropEmpty').textContent = 'Historical balance data is unavailable.'; }
    $('opsUpdated').textContent = 'Updated ' + new Date().toLocaleTimeString();
  }

  async function refresh() {
    if (loading) return;
    loading = true;
    try { render(await api('GET')); setStatus('Connected · live operator snapshot loaded.', 'ok'); }
    catch (error) { setStatus(error.message, 'error'); }
    finally { loading = false; }
  }

  async function update(body, input) {
    input.disabled = true;
    try { const payload = await api('POST', body); renderFeatures(payload.features); renderExperiment(payload.reward_experiment); setStatus('Runtime configuration updated.', 'ok'); }
    catch (error) { setStatus(error.message, 'error'); await refresh(); }
    finally { input.disabled = false; }
  }

  $('opsAuth').addEventListener('submit', event => { event.preventDefault(); token = $('opsToken').value.trim(); sessionStorage.setItem('abyss_ops_token', token); refresh(); });
  $('opsRefresh').addEventListener('click', refresh);
  root.querySelectorAll('[data-ops-feature]').forEach(input => input.addEventListener('change', () => update({ feature: input.dataset.opsFeature, enabled: input.checked }, input)));
  root.querySelectorAll('[data-ops-percent]').forEach(input => { input.addEventListener('input', () => { $(input.id + 'Value').textContent = input.value + '%'; }); input.addEventListener('change', () => update({ feature: input.dataset.opsPercent, percent: Number(input.value) }, input)); });
  $('rewardBonus').addEventListener('input', () => { $('rewardBonusValue').textContent = '+' + (Number($('rewardBonus').value) / 100).toFixed(2) + '%'; });
  $('rewardBonus').addEventListener('change', event => update({ feature: event.target.dataset.opsBonus, bonus_bps: Number(event.target.value) }, event.target));
  if (token) { $('opsToken').value = token; refresh(); }
  window.setInterval(() => { if (token && document.visibilityState === 'visible') refresh(); }, 30000);
}());
