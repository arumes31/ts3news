(function () {
  'use strict';
  const $ = id => document.getElementById(id);
  if (!$('abyssTalentCanvas')) return;
  const NS = 'http://www.w3.org/2000/svg';
  const geometry = { width: 840, height: 1100, foundationY: 92, subclassY: 628, step: 76 };
  const viewport = $('abyssTalentViewport');
  const canvas = $('abyssTalentCanvas');
  const connections = $('abyssTalentConnections');
  let data, cls, sub, lastData, revision = -1, saving = false;
  let foundations = {}, specializations = {}, positions = new Map();
  let zoom = 1, inspected = null, tooltipAnchor = null, tooltipTimer, dragging = null;
  const tooltip = document.createElement('div');
  tooltip.id = 'abyssClassTalentTooltip';
  tooltip.className = 'ab-talent-tooltip';
  tooltip.setAttribute('role', 'tooltip');
  tooltip.hidden = true;
  document.body.append(tooltip);

  function el(tag, text, className) {
    const node = document.createElement(tag);
    if (text !== undefined) node.textContent = text;
    if (className) node.className = className;
    return node;
  }
  const idsFor = tree => tree.subclass ? (specializations[tree.id] || []) : (foundations[tree.id] || []);
  const nodeFor = (tree, id) => tree.nodes.find(node => node.id === id);
  const budget = tree => tree.subclass ? Math.max(0, data.class_progress[tree.class].points - 5) : Math.min(5, data.class_progress[tree.class].points);
  const inspect = text => { $('abyssTalentInspect').textContent = text; };
  const treeName = tree => tree.subclass ? cls.subclasses.find(choice => choice.id === tree.id).name : cls.name;

  function lockedReason(tree) {
    if (saving) return 'Saving your path. Wait for the result before making another change.';
    if (data.locked) return 'Talents are locked during this run. Bank or end it first.';
    if (tree.subclass && sub.id !== tree.id) return 'Choose ' + treeName(tree) + ' to edit this branch.';
    if (tree.subclass && (foundations[cls.id] || []).length !== 5) return 'Choose all five foundation tiers first.';
    return '';
  }
  function canAdd(tree, node, ids) {
    const locked = lockedReason(tree);
    if (locked) return locked;
    const chosen = ids.map(id => nodeFor(tree, id)).filter(Boolean);
    const same = chosen.filter(choice => choice.tier === node.tier);
    const limit = tree.subclass && node.tier < 5 ? 2 : 1;
    if (same.length >= limit && limit === 2) return 'Choose at most two talents in this tier. Remove one to change the choice.';
    const replacing = limit === 1 ? same.length : 0;
    if (ids.length - replacing >= budget(tree)) return 'Earn the next class talent point from Abyss monster kills.';
    if (node.tier > 0 && !chosen.some(choice => choice.tier === node.tier - 1)) return 'Choose a connected talent in the previous tier first.';
    if (node.tier === 5 && chosen.filter(choice => choice.tier < 5).length < 9) return 'Choose nine subclass talents before selecting a final talent.';
    return '';
  }
  function reasonFor(tree, node) {
    return lockedReason(tree) || (idsFor(tree).includes(node.id) ? '' : canAdd(tree, node, idsFor(tree)));
  }
  function prune(tree, ids) {
    let kept = ids.filter(id => nodeFor(tree, id));
    for (let tier = 1; tier < 6; tier++) {
      if (!kept.some(id => nodeFor(tree, id).tier === tier - 1)) kept = kept.filter(id => nodeFor(tree, id).tier < tier);
    }
    if (tree.subclass && kept.filter(id => nodeFor(tree, id).tier < 5).length < 9) kept = kept.filter(id => nodeFor(tree, id).tier < 5);
    return kept;
  }
  function toggle(tree, node) {
    const reason = reasonFor(tree, node);
    if (reason) { showDetails(tree, node); inspect(node.name + '. ' + reason); return; }
    let chosen = idsFor(tree).slice();
    if (chosen.includes(node.id)) chosen = prune(tree, chosen.filter(id => id !== node.id));
    else {
      if (!tree.subclass || node.tier === 5) chosen = chosen.filter(id => nodeFor(tree, id).tier !== node.tier);
      chosen.push(node.id);
    }
    if (tree.subclass) specializations[tree.id] = chosen;
    else foundations[tree.id] = chosen;
    draw();
    canvas.querySelector('[data-talent="' + node.id + '"]').focus({ preventScroll: true });
    showDetails(tree, node);
    inspect(node.name + '. ' + node.description + ' Unsaved path changes.');
  }
  function requirement(tree, node) {
    const reason = reasonFor(tree, node);
    if (reason) return reason;
    if (idsFor(tree).includes(node.id)) return 'Learned in this path. Removing a prerequisite also removes its dependent choices.';
    const replacing = (!tree.subclass || node.tier === 5) && idsFor(tree).some(id => nodeFor(tree, id).tier === node.tier);
    return replacing ? 'Replaces your other choice in this tier. No additional point required.' : 'Costs 1 class talent point. Save your path to apply this choice.';
  }
  function showDetails(tree, node) {
    inspected = { tree, node };
    const chosen = idsFor(tree).includes(node.id);
    const art = $('abyssTalentDetailArt');
    art.src = node.art; art.hidden = false;
    $('abyssTalentDetailTitle').textContent = node.name.split(' · ').pop();
    $('abyssTalentDetailKind').textContent = treeName(tree) + (tree.subclass ? ' extension' : ' foundation') + ' · ' + (node.tier === 5 ? 'Final talent' : 'Tier ' + (node.tier + 1)) + ' · ' + (chosen ? '1/1' : '0/1');
    $('abyssTalentDetailEffect').textContent = node.description;
    $('abyssTalentDetailRequirement').textContent = requirement(tree, node);
    $('abyssTalentDetailAction').textContent = chosen ? 'Remove talent' : 'Choose talent';
    $('abyssTalentDetailAction').disabled = !!reasonFor(tree, node);
  }
  function hideTooltip() {
    clearTimeout(tooltipTimer);
    if (tooltipAnchor) tooltipAnchor.removeAttribute('aria-describedby');
    tooltipAnchor = null;
    tooltip.hidden = true;
  }
  function positionTooltip() {
    if (!tooltipAnchor || tooltip.hidden) return;
    const anchor = tooltipAnchor.getBoundingClientRect();
    const bounds = viewport.getBoundingClientRect();
    if (anchor.bottom < bounds.top || anchor.top > bounds.bottom || anchor.right < bounds.left || anchor.left > bounds.right) { hideTooltip(); return; }
    // Keep the floating details beside the graph so they never cover another node.
    const inspector = document.querySelector('.ab-talent-inspector').getBoundingClientRect();
    tooltip.style.left = Math.max(8, Math.min(inspector.left, document.documentElement.clientWidth - tooltip.offsetWidth - 8)) + 'px';
    tooltip.style.top = Math.max(8, Math.min(anchor.top, window.innerHeight - tooltip.offsetHeight - 8)) + 'px';
  }
  function showTooltip(tree, node, button) {
    clearTimeout(tooltipTimer);
    showDetails(tree, node);
    if (window.matchMedia('(max-width: 760px)').matches) return;
    hideTooltip();
    tooltip.append(el('h4', node.name), el('p', node.description, 'ab-talent-tooltip-effect'), el('p', requirement(tree, node)), el('small', treeName(tree) + ' · Rank ' + (idsFor(tree).includes(node.id) ? '1/1' : '0/1')));
    tooltipAnchor = button;
    button.setAttribute('aria-describedby', tooltip.id);
    tooltip.hidden = false;
    positionTooltip();
  }
  function preview(tree, node, button) {
    tooltip.replaceChildren();
    showTooltip(tree, node, button);
  }
  function scheduleHide() { clearTimeout(tooltipTimer); tooltipTimer = setTimeout(hideTooltip, 140); }

  function point(tree, node) {
    const isSub = !!tree.subclass;
    const center = isSub ? (cls.subclasses.findIndex(choice => choice.id === tree.id) === 0 ? 210 : 630) : 420;
    return { x: center + (node.branch - 1) * (isSub ? 106 : 116), y: (isSub ? geometry.subclassY : geometry.foundationY) + 30 + node.tier * geometry.step };
  }
  function edge(from, to, active, available, attributes) {
    const path = document.createElementNS(NS, 'path');
    path.setAttribute('d', 'M ' + from.x + ' ' + (from.y + 25) + ' L ' + to.x + ' ' + (to.y - 29));
    path.setAttribute('class', 'ab-talent-edge' + (active ? ' is-learned' : available ? ' is-available' : ''));
    path.setAttribute('marker-end', active ? 'url(#abTalentArrowLearned)' : 'url(#abTalentArrow)');
    Object.entries(attributes || {}).forEach(([key, value]) => path.setAttribute(key, value));
    connections.append(path);
  }
  function drawConnections() {
    connections.replaceChildren();
    connections.setAttribute('viewBox', '0 0 ' + geometry.width + ' ' + geometry.height);
    const defs = document.createElementNS(NS, 'defs');
    for (const [id, color] of [['abTalentArrow', '#526174'], ['abTalentArrowLearned', '#e7c47c']]) {
      const marker = document.createElementNS(NS, 'marker');
      for (const [key, value] of Object.entries({ id, viewBox: '0 0 8 8', refX: '7', refY: '4', markerWidth: '5', markerHeight: '5', orient: 'auto-start-reverse' })) marker.setAttribute(key, value);
      const arrow = document.createElementNS(NS, 'path');
      arrow.setAttribute('d', 'M 0 0 L 8 4 L 0 8 Z'); arrow.setAttribute('fill', color); marker.append(arrow); defs.append(marker);
    }
    connections.append(defs);
    const base = data.talent_catalog[cls.id];
    for (const tree of [base, ...cls.subclasses.map(choice => data.talent_catalog[choice.id])]) {
      const ids = idsFor(tree);
      for (const node of tree.nodes) {
        const end = point(tree, node);
        if (node.tier === 0) {
          const start = tree.subclass ? { x: end.x - (node.branch - 1) * 106, y: 555 } : { x: 420, y: 42 };
          edge(start, end, ids.includes(node.id) && !lockedReason(tree), !reasonFor(tree, node));
        } else {
          for (const parent of tree.nodes.filter(candidate => candidate.tier === node.tier - 1)) {
            edge(point(tree, parent), end, ids.includes(parent.id) && ids.includes(node.id) && (!tree.subclass || sub.id === tree.id), ids.includes(parent.id) && !reasonFor(tree, node));
          }
        }
      }
    }
    cls.subclasses.forEach((choice, index) => {
      for (const parent of base.nodes.filter(node => node.tier === 4)) {
        edge(point(base, parent), { x: index === 0 ? 210 : 630, y: 543 }, idsFor(base).includes(parent.id) && sub.id === choice.id && idsFor(base).length === 5, idsFor(base).length === 5, { 'data-extends': cls.id, 'data-branch': choice.id });
      }
    });
  }
  function drawTree(tree, target) {
    target.replaceChildren();
    target.dataset.tree = tree.id;
    target.setAttribute('role', 'group');
    target.setAttribute('aria-label', treeName(tree) + (tree.subclass ? ' subclass extension' : ' class foundation'));
    const chosen = idsFor(tree);
    tree.nodes.forEach(node => {
      const position = point(tree, node);
      positions.set(node.id, { ...position, tree, node });
      const button = el('button', undefined, 'ab-talent-node' + (node.tier === 5 ? ' is-capstone' : ''));
      button.type = 'button';
      button.dataset.talent = node.id;
      button.style.left = position.x + 'px';
      button.style.top = (position.y - (tree.subclass ? geometry.subclassY : geometry.foundationY)) + 'px';
      button.setAttribute('aria-pressed', String(chosen.includes(node.id)));
      const reason = reasonFor(tree, node);
      button.setAttribute('aria-disabled', String(!!reason));
      button.setAttribute('aria-label', node.name + '. ' + node.description + ' Rank ' + (chosen.includes(node.id) ? '1 of 1.' : '0 of 1.') + (reason ? ' ' + reason : ''));
      const image = el('img'); image.src = node.art; image.alt = ''; image.width = 44; image.height = 44; image.loading = 'lazy';
      button.append(image, el('span', chosen.includes(node.id) ? '1/1' : '0/1', 'ab-talent-rank'), el('strong', node.name.split(' · ').pop(), 'ab-talent-name'));
      button.addEventListener('pointerenter', event => { if (event.pointerType !== 'touch') preview(tree, node, button); });
      button.addEventListener('pointerleave', scheduleHide);
      button.addEventListener('focus', () => preview(tree, node, button));
      button.addEventListener('blur', scheduleHide);
      button.addEventListener('click', () => { hideTooltip(); toggle(tree, node); });
      target.append(button);
    });
  }
  function selectBranch(id) {
    const select = $('abyssSubclass');
    select.value = id;
    select.dispatchEvent(new Event('change'));
  }
  function drawGate(foundation) {
    const gate = $('abyssSubclassGate'); gate.replaceChildren();
    const message = el('p', foundation.length === 5 ? 'Foundation complete · extend into one subclass' : 'Choose all five foundation tiers to unlock a subclass', 'ab-talent-gate-status');
    gate.append(message);
    cls.subclasses.forEach((choice, index) => {
      const button = el('button', undefined, 'ab-talent-subclass-choice');
      button.type = 'button'; button.dataset.subclassChoice = choice.id;
      button.style.left = (index === 0 ? 210 : 630) + 'px';
      button.setAttribute('aria-pressed', String(sub.id === choice.id));
      const art = el('span', undefined, 'ab-talent-branch-art');
      const frame = data.catalog.findIndex(candidate => candidate.id === cls.id) * 2 + index;
      art.style.backgroundImage = 'url("/static/abyss_subclasses_' + (frame < 6 ? 'martial' : 'mystic') + '_v1.png")';
      art.style.backgroundPosition = '0% ' + (frame % 6 * 20) + '%';
      art.setAttribute('aria-hidden', 'true');
      const text = el('span');
      text.append(el('strong', choice.name), el('small', sub.id === choice.id ? 'Editing this path' : foundation.length === 5 ? 'Inspect this branch' : 'Unlock after foundation'));
      button.append(art, text); button.addEventListener('click', () => selectBranch(choice.id)); gate.append(button);
    });
  }
  function updateZoom(next, fit) {
    const before = zoom;
    const centerX = (viewport.scrollLeft + viewport.clientWidth / 2) / before;
    const centerY = (viewport.scrollTop + viewport.clientHeight / 2) / before;
    zoom = Math.max(.2, Math.min(1.5, next));
    canvas.style.transform = 'scale(' + zoom + ')';
    $('abyssTalentSizer').style.width = geometry.width * zoom + 'px';
    $('abyssTalentSizer').style.height = geometry.height * zoom + 'px';
    $('abyssTalentZoom').textContent = Math.round(zoom * 100) + '%';
    $('abyssTalentZoomOut').disabled = zoom <= .2;
    $('abyssTalentZoomIn').disabled = zoom >= 1.5;
    viewport.scrollLeft = fit ? 0 : centerX * zoom - viewport.clientWidth / 2;
    viewport.scrollTop = fit ? 0 : centerY * zoom - viewport.clientHeight / 2;
    hideTooltip();
  }
  function draw() {
    if (!data || !data.talent_catalog) return;
    hideTooltip();
    positions = new Map();
    const progress = data.class_progress[cls.id];
    const foundation = foundations[cls.id] || [];
    const isSub = sub.id !== cls.id;
    const selected = isSub ? specializations[sub.id] || [] : [];
    $('abyssTalentTitle').textContent = cls.name + ' talent tree';
    $('abyssTalentBudget').textContent = progress.points + ' / 15 points earned · ' + foundation.length + '/5 foundation' + (isSub ? ' · ' + selected.length + '/10 subclass' : '');
    $('abyssTalentPath').textContent = cls.name + (isSub ? ' → ' + sub.name : ' → choose a subclass');
    const xp = (progress.xp / 1000).toLocaleString(undefined, { maximumFractionDigits: 3 });
    $('abyssTalentProgress').textContent = xp + ' class XP · ' + progress.clears + ' combat floors cleared with this class.' + (progress.next_xp ? ' Next point at ' + (progress.next_xp / 1000).toLocaleString() + ' class XP.' : ' All fifteen points earned.');
    $('abyssTalentXP').max = progress.next_xp || Math.max(1, progress.xp); $('abyssTalentXP').value = progress.xp;
    $('abyssTalentPacing').textContent = (progress.legacy_credit ? 'Existing subclass preserved: the first five points were credited once. ' : '') + 'First five points: 1, 5, 15, 35 and 75 class XP. Later points take 500–5,000 more qualifying combat floors. XP is permanent; a normal complete combat floor grants 1 XP, bosses up to 10% extra.' + (progress.points >= 5 && progress.best_depth >= 100 ? ' Floors below ' + Math.floor(progress.best_depth / 4) + ' now grant 25% class XP.' : '');
    const root = $('abyssTalentRoot');
    root.replaceChildren();
    const portrait = el('span', undefined, 'ab-talent-root-art');
    portrait.style.backgroundPosition = '0% ' + data.catalog.findIndex(candidate => candidate.id === cls.id) * 20 + '%';
    portrait.setAttribute('aria-hidden', 'true');
    root.append(portrait, el('strong', cls.name), el('small', foundation.length + '/5 foundation talents'));
    drawTree(data.talent_catalog[cls.id], $('abyssFoundationTree'));
    drawGate(foundation);
    const extensions = $('abyssSubclassTree'); extensions.hidden = false; extensions.replaceChildren();
    cls.subclasses.forEach(choice => {
      const branch = el('div', undefined, 'ab-talent-branch' + (sub.id === choice.id ? ' is-current' : ' is-inactive'));
      drawTree(data.talent_catalog[choice.id], branch);
      extensions.append(branch);
    });
    drawConnections();
    $('abyssTalentSave').disabled = data.locked || saving;
    $('abyssTalentReset').disabled = data.locked || saving;
    if (inspected && positions.has(inspected.node.id)) {
      const entry = positions.get(inspected.node.id); showDetails(entry.tree, entry.node);
    }
  }
  window.AbyssClassTalents = {
    render: function (next, chosenClass, chosenSub) {
      const changedClass = !cls || cls.id !== chosenClass.id;
      data = next; cls = chosenClass; sub = chosenSub;
      if (revision !== next.state.revision || lastData !== next) {
        foundations = {}; specializations = {};
        for (const candidate of next.catalog) foundations[candidate.id] = [...(next.class_progress[candidate.id].foundation || [])];
        for (const [id, profile] of Object.entries(next.state.profiles)) specializations[id] = [...(profile.talents || [])];
        revision = next.state.revision; lastData = next;
      }
      if (changedClass) inspected = null;
      draw();
      if (changedClass) {
        showDetails(data.talent_catalog[cls.id], data.talent_catalog[cls.id].nodes[0]);
        viewport.scrollTop = 0;
        viewport.scrollLeft = Math.max(0, geometry.width * zoom / 2 - viewport.clientWidth / 2);
      }
    }
  };
  $('abyssTalentDetailAction').addEventListener('click', () => { if (inspected) toggle(inspected.tree, inspected.node); });
  $('abyssTalentReset').addEventListener('click', () => {
    if (!data || data.locked || saving) return;
    if (sub.id === cls.id) foundations[cls.id] = [];
    else specializations[sub.id] = [];
    draw(); inspect('Path reset in this preview. Save to apply it; earned XP is preserved.');
  });
  $('abyssTalentSave').addEventListener('click', async () => {
    if (!data || data.locked || saving) return;
    const foundation = foundations[cls.id] || [];
    const selected = sub.id !== cls.id && foundation.length === 5 ? sub.id : '';
    const body = { class: cls.id, selected, foundation };
    if (selected) {
      const profile = data.state.profiles[selected] || {};
      body.profile = {
        skills: profile.skills ?? data.skills.filter(skill => skill.source !== 'class_signature' && data.learned.some(learned => learned.id === skill.id)).map(skill => skill.id),
        pins: profile.pins || [], talents: specializations[selected] || []
      };
    }
    saving = true; draw();
    try { await window.AbyssClassWorkbench.saveTalents(body); }
    finally { saving = false; draw(); inspect($('abyssClassStatus').textContent); }
  });
  $('abyssTalentZoomIn').addEventListener('click', () => updateZoom(zoom + .15));
  $('abyssTalentZoomOut').addEventListener('click', () => updateZoom(zoom - .15));
  $('abyssTalentFit').addEventListener('click', () => updateZoom(Math.min((viewport.clientWidth - 4) / geometry.width, (viewport.clientHeight - 4) / geometry.height, 1), true));
  tooltip.addEventListener('pointerenter', () => clearTimeout(tooltipTimer));
  tooltip.addEventListener('pointerleave', scheduleHide);
  document.addEventListener('keydown', event => { if (event.key === 'Escape') hideTooltip(); });
  window.addEventListener('scroll', positionTooltip, { capture: true, passive: true });
  window.addEventListener('resize', hideTooltip);
  viewport.addEventListener('pointerdown', event => {
    if (event.pointerType !== 'mouse' || event.button !== 0 || event.target.closest('button')) return;
    dragging = { x: event.clientX, y: event.clientY, left: viewport.scrollLeft, top: viewport.scrollTop };
    viewport.setPointerCapture(event.pointerId); viewport.classList.add('is-panning'); hideTooltip();
  });
  viewport.addEventListener('pointermove', event => {
    if (!dragging) return;
    viewport.scrollLeft = dragging.left - (event.clientX - dragging.x);
    viewport.scrollTop = dragging.top - (event.clientY - dragging.y);
  });
  function endDrag() { dragging = null; viewport.classList.remove('is-panning'); }
  viewport.addEventListener('pointerup', endDrag);
  viewport.addEventListener('pointercancel', endDrag);
  updateZoom(1, true);
  const initialView = new ResizeObserver(() => {
    if (!viewport.clientWidth) return;
    if (window.matchMedia('(min-width: 761px)').matches) {
      const fit = Math.min((viewport.clientWidth - 4) / geometry.width, (viewport.clientHeight - 4) / geometry.height, 1);
      updateZoom(Math.max(.6, fit), true);
    }
    viewport.scrollLeft = Math.max(0, geometry.width * zoom / 2 - viewport.clientWidth / 2);
    initialView.disconnect();
  });
  initialView.observe(viewport);
})();
