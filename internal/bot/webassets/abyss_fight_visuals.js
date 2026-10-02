/* Presentation only: no action requests, inferred damage, or combat-state writes. */
(function (global) {
  'use strict';
  function finite(value, fallback) { return typeof value === 'number' && Number.isFinite(value) ? value : fallback; }
  function clamp(value, low, high) { return Math.max(low, Math.min(high, value)); }
  function preferences(input) {
    var p = input && typeof input === 'object' ? input : {};
    return {
      quality: ['auto', 'low', 'full'].includes(p.quality) ? p.quality : 'auto',
      contrast: p.contrast === 'high' ? 'high' : 'normal', numbers: p.numbers === 'large' ? 'large' : 'normal',
      particles: typeof p.particles === 'boolean' ? p.particles : true,
      backdrop: clamp(finite(p.backdrop, 45), 0, 100), linger: clamp(finite(p.linger, 900), 600, 1800)
    };
  }
  function health(unit) {
    unit = unit || {};
    if (unit.hp_hidden || !(finite(unit.max_hp, 0) > 0) || !Number.isFinite(unit.hp)) return {kind: 'concealed', percent: null, label: 'Health concealed'};
    var percent = clamp(unit.hp / unit.max_hp * 100, 0, 100);
    var kind = percent === 0 ? 'defeated' : percent <= 20 ? 'critical' : percent <= 50 ? 'hurt' : 'healthy';
    return {kind: kind, percent: percent, label: {defeated: 'Defeated', critical: 'Critical', hurt: 'Wounded', healthy: 'Healthy'}[kind]};
  }
  function impact(target) {
    var t = target || {};
    return t.status === 'revived' ? 'revive' : t.dodged ? 'dodge' : t.blocked ? 'block' : t.absorbed > 0 ? 'absorb' : t.healing > 0 ? 'heal' : t.critical && t.damage > 0 ? 'critical' : t.damage > 0 ? 'damage' : t.status ? 'status' : 'none';
  }
  function position(point, width, height, lane) {
    point = point || {}; width = Math.max(0, finite(width, 0)); height = Math.max(0, finite(height, 0));
    var mx = Math.min(48, width / 2), my = Math.min(12, height / 2);
    return {x: clamp(finite(point.x, width / 2), mx, width - mx), y: clamp(finite(point.y, height / 2) - clamp(finite(lane, 0), 0, 8) * 24, my, height - my)};
  }
  function sequence(value) { return Number.isSafeInteger(value) && value > 0 ? value : 0; }
  var api = global.AbyssFightVisuals = {model: {preferences: preferences, health: health, impact: impact, position: position, sequence: sequence}};
  if (typeof document === 'undefined') return;

  var host, settings, session = '', lastRound = 0, lastPhase = '', observed = 0, motion, structured = false;
  var timers = new Set(), playing = new Set(), atlasLoaded = false;
  var prefs = preferences();
  try { prefs = preferences(JSON.parse(localStorage.getItem('abyssFightVisuals') || '{}')); } catch (_) {}
  function el(tag, text, cls) { var node = document.createElement(tag); if (text != null) node.textContent = text; if (cls) node.className = cls; return node; }
  function byID(id) { return document.getElementById(id); }
  function save() { try { localStorage.setItem('abyssFightVisuals', JSON.stringify(prefs)); } catch (_) {} }
  function reduced() { return !!(motion && motion.matches || host && host.dataset.effects === 'reduced'); }
  function quality() {
    if (prefs.quality !== 'auto') return prefs.quality;
    return global.innerWidth <= 760 || global.navigator && (global.navigator.hardwareConcurrency <= 4 || global.navigator.connection && global.navigator.connection.saveData) ? 'low' : 'full';
  }
  function after(fn, ms) { var id = setTimeout(function () { timers.delete(id); fn(); }, ms); timers.add(id); return id; }
  function move(node, frames, ms) {
    if (reduced() || !node || !node.animate) return;
    try {
      var animation = node.animate(frames, {duration: ms, easing: 'ease-out'}); playing.add(animation);
      animation.finished.then(function () { playing.delete(animation); }, function () { playing.delete(animation); });
    } catch (_) { /* Static feedback remains available on older renderers. */ }
  }
  function clear() {
    timers.forEach(clearTimeout); timers.clear(); playing.forEach(function (a) { a.cancel(); }); playing.clear();
    if (host) {
      host.querySelectorAll('.ab-fight-flourish,.ab-hp-trail').forEach(function (n) { n.remove(); });
      host.querySelectorAll('.ab-fight-target').forEach(function (n) { n.classList.remove('ab-fight-target'); });
    }
  }
  function apply() {
    if (!host) return;
    host.dataset.visualQuality = quality(); host.dataset.visualContrast = prefs.contrast;
    host.dataset.visualNumbers = prefs.numbers; host.dataset.visualParticles = prefs.particles ? 'on' : 'off';
    host.style.setProperty('--fight-backdrop', String(prefs.backdrop / 100));
    var note = byID('liveVisualMotionNote');
    if (note) note.textContent = reduced() ? 'Reduced motion active: static feedback, no moving particles. Combat continues normally.' : 'Visual changes affect presentation only. Combat timing and targeting stay live.';
    if (reduced() || !prefs.particles || quality() === 'low') clear();
  }
  function control(id, label, choices, value) {
    var node = el(choices ? 'select' : 'input'); node.id = id;
    if (choices) choices.forEach(function (pair) { var option = el('option', pair[1]); option.value = pair[0]; node.appendChild(option); });
    node.value = value; var wrapper = el('label', label); wrapper.htmlFor = id; wrapper.appendChild(node); settings.appendChild(wrapper); return node;
  }
  function syncControls() {
    var values = {liveVisualQuality: prefs.quality, liveVisualContrast: prefs.contrast, liveVisualNumbers: prefs.numbers, liveVisualBackdrop: prefs.backdrop, liveVisualLinger: prefs.linger};
    Object.keys(values).forEach(function (id) { byID(id).value = values[id]; }); byID('liveVisualParticles').checked = prefs.particles;
  }
  function preload() {
    if (atlasLoaded || !global.Image) return; atlasLoaded = true;
    ['roles', 'creatures', 'bestiary', 'bosses'].forEach(function (name) {
      var image = new global.Image(); image.decoding = 'async';
      image.src = '/static/abyss_combat_' + name + '_v2.png' + (global.__ASSET_VER__ ? '?v=' + encodeURIComponent(global.__ASSET_VER__) : '');
      if (image.decode) image.decode().catch(function () {});
    });
  }
  function init(stage) {
    if (host === stage || !stage) return; host = stage; motion = global.matchMedia('(prefers-reduced-motion: reduce)');
    var scenery = el('div', null, 'ab-fight-scenery'); scenery.setAttribute('aria-hidden', 'true'); host.prepend(scenery);
    var panel = el('div', null, 'ab-fight-presentation'), heading = el('div', null, 'ab-fight-presentation-heading');
    var caption = el('span', 'Ready', ''); caption.id = 'liveVisualCaption';
    var playback = el('span', 'Live'); playback.id = 'liveVisualPlayback'; playback.setAttribute('aria-live', 'off');
    var skip = el('button', 'Skip effects', 'ab-live-tool'); skip.type = 'button'; skip.id = 'liveAnimationSkip'; skip.disabled = true;
    skip.title = 'Finish queued visual effects. Combat and action deadlines continue.';
    skip.onclick = function () { if (global.AbyssCombatAnimation) global.AbyssCombatAnimation.skip(); };
    heading.append(caption, playback, skip); panel.appendChild(heading); host.appendChild(panel);
    settings = el('div', null, 'ab-fight-options'); settings.setAttribute('role', 'group'); settings.setAttribute('aria-label', 'Fight graphics');
    var settingsHost = document.querySelector('#liveSettingsDrawer .ab-animation-settings'); if (!settingsHost) return; settingsHost.appendChild(settings);
    control('liveVisualQuality', 'Graphics quality', [['auto','Automatic'],['full','Full'],['low','Low effects']], prefs.quality);
    control('liveVisualContrast', 'Scene contrast', [['normal','Normal'],['high','High contrast']], prefs.contrast);
    control('liveVisualNumbers', 'Text size', [['normal','Normal'],['large','Large numbers and labels']], prefs.numbers);
    var particle = control('liveVisualParticles', 'Impact particles', null, ''); particle.type = 'checkbox'; particle.checked = prefs.particles; particle.parentElement.className = 'ab-visual-check';
    var backdrop = control('liveVisualBackdrop', 'Backdrop strength', null, prefs.backdrop); backdrop.type = 'range'; backdrop.min = '0'; backdrop.max = '100'; backdrop.step = '5';
    var linger = control('liveVisualLinger', 'Number duration (milliseconds)', null, prefs.linger); linger.type = 'range'; linger.min = '600'; linger.max = '1800'; linger.step = '100';
    var note = el('small'); note.id = 'liveVisualMotionNote'; note.setAttribute('role', 'status'); settings.appendChild(note);
    var reset = el('button', 'Reset fight graphics', 'ab-live-tool'); reset.id = 'liveVisualReset'; reset.type = 'button'; settings.appendChild(reset);
    reset.onclick = function () { prefs = preferences(); syncControls(); save(); apply(); fit(); };
    settings.addEventListener('input', function (event) {
      if (!event.target.matches('input,select')) return;
      prefs = preferences({quality: byID('liveVisualQuality').value, contrast: byID('liveVisualContrast').value, numbers: byID('liveVisualNumbers').value, particles: particle.checked, backdrop: Number(backdrop.value), linger: Number(linger.value)}); save(); apply(); fit();
    });
    var history = el('details', null, 'ab-fight-history'); history.appendChild(el('summary', 'Recent animation events'));
    var list = el('ol'); list.id = 'liveVisualHistory'; list.setAttribute('aria-label', 'Recent confirmed combat events'); history.appendChild(list); settingsHost.after(history);
    if (motion.addEventListener) motion.addEventListener('change', apply);
    global.addEventListener('resize', function () { clear(); apply(); fit(); }, {passive: true});
    if (global.ResizeObserver) new global.ResizeObserver(fit).observe(host);
    document.addEventListener('visibilitychange', function () { if (document.hidden) clear(); });
    global.addEventListener('pagehide', clear);
    byID('liveAnimationEffects').addEventListener('change', apply);
    apply(); preload();
  }
  function snapshot(state) {
    init(byID('livePixelStage')); if (!host) return;
    structured = Array.isArray(state.presentation_events) || state.presentation_cursor != null;
    if (session !== state.session_id) {
      clear(); session = state.session_id; observed = 0; lastRound = Number(state.round) || 0; lastPhase = '';
      byID('liveVisualHistory').replaceChildren(); byID('liveVisualCaption').textContent = 'Ready';
      host.querySelectorAll('.ab-fight-result').forEach(function (n) { n.remove(); });
    }
    var phase = String(state.phase || 'planning'); host.dataset.visualPhase = phase;
    if (phase !== lastPhase) {
      byID('liveVisualCaption').textContent = phase === 'planning' ? 'Choose your next action' : phase.replace(/_/g, ' ');
      if (phase === 'complete' || phase === 'failed') {
        host.querySelectorAll('.ab-fight-result').forEach(function (n) { n.remove(); });
        var result = el('div', phase === 'complete' ? 'Fight complete' : 'Fight ended', 'ab-fight-result ' + phase); result.setAttribute('role', 'status'); host.appendChild(result);
      }
      lastPhase = phase;
    }
    var biome = String(state.biome || host.dataset.biomeArt || '').toLowerCase();
    var accent = /fire|lava|ember/.test(biome) ? '#e49a6b' : /frost|ice|snow/.test(biome) ? '#96cbdc' : /swamp|forest|moss/.test(biome) ? '#a3c093' : /void|shadow/.test(biome) ? '#bbacd8' : '#c9ad73';
    host.style.setProperty('--fight-accent', accent); apply();
  }
  function center(actor) {
    var r = host.getBoundingClientRect(), sprite = actor.querySelector('.ab-actor-sprite'), a = (sprite || actor).getBoundingClientRect();
    return {x: a.left - r.left + a.width / 2, y: a.top - r.top + a.height * .6};
  }
  function flourish(actor, kind, color, duration) {
    if (!host || !actor || reduced() || !prefs.particles || kind === 'none') return;
    var cap = quality() === 'low' ? 8 : 36, old = host.querySelectorAll('.ab-fight-flourish');
    if (old.length >= cap) old[0].remove();
    var point = position(center(actor), host.clientWidth, host.clientHeight, 0), node = el('span', null, 'ab-fight-flourish ' + kind);
    node.setAttribute('aria-hidden', 'true'); node.dataset.targetId = actor.dataset.entityId;
    if (color) node.style.setProperty('--fight-fx', color);
    node.style.left = point.x + 'px'; node.style.top = point.y + 'px'; host.appendChild(node);
    move(node, [{opacity: .8, transform: 'translate(-50%,-50%) scale(.55)'}, {opacity: 0, transform: 'translate(-50%,-60%) scale(1.3)'}], duration || 350);
    after(function () { node.remove(); }, duration || 350);
  }
  function actor(node, unit) {
    if (!host) init(byID('livePixelStage')); if (!host) return;
    var hp = health(unit), previous = node._fightHealth, info = node.querySelector('.ab-combat-unit-info');
    node.dataset.health = hp.kind; node.title = String(unit.name || '') + ' · ' + String(unit.role || (unit.is_self ? 'You' : unit.is_player ? 'Ally' : 'Combatant'));
    var label = info.querySelector('.ab-health-state'); if (!label) { label = el('span', '', 'ab-health-state'); info.appendChild(label); }
    label.textContent = hp.kind === 'healthy' ? '' : hp.label;
    var name = info.querySelector('.ab-pixel-name b');
    if (unit.is_self && name && !name.querySelector('.ab-fight-self')) name.appendChild(el('span', 'YOU', 'ab-fight-self'));
    var bar = info.querySelector('.ab-overhead-hp');
    if (bar) {
      bar.setAttribute('role', 'img');
      bar.setAttribute('aria-label', hp.kind === 'concealed' ? 'Health concealed' : 'Health ' + unit.hp + ' / ' + unit.max_hp + ', ' + hp.label);
      if (hp.kind === 'concealed') { bar.classList.add('concealed'); bar.querySelector('em').textContent = '??'; bar.querySelector('i').style.width = '100%'; }
      if (!structured && previous && previous.percent != null && hp.percent != null && previous.percent !== hp.percent && !reduced()) {
        var trail = el('span', null, 'ab-hp-trail' + (hp.percent > previous.percent ? ' heal' : '')); trail.setAttribute('aria-hidden', 'true'); trail.style.width = hp.percent + '%'; bar.prepend(trail);
        move(trail, [{width: previous.percent + '%'}, {width: hp.percent + '%'}], 500); after(function () { trail.remove(); }, 510);
      }
    }
    var shield = Math.max(0, finite(unit.shield, 0)); node.style.setProperty('--fight-shield', String(unit.max_shield > 0 ? clamp(shield / unit.max_shield, 0, 1) : 0));
    if (!structured && node._fightShield > 0 && shield === 0 && !unit.hp_hidden) flourish(node, 'absorb', '#bddcff', 260);
    node._fightShield = shield; node._fightHealth = hp;
    // Status visibility follows the snapshot, including effects beyond the four tiny chips.
    var statuses = (unit.effects || []).filter(function (effect) { return !effect.affix; });
    var active = hp.kind !== 'defeated' ? statuses.map(function (effect) { return String(effect.key || '') + ' ' + String(effect.name || ''); }).join(' ').toLowerCase() : '';
    var dotHost = node.querySelector('.ab-dot-statuses');
    if (!dotHost) { dotHost = el('span', null, 'ab-dot-statuses'); node.appendChild(dotHost); }
    var labels = [];
    ['poison', 'bleed'].forEach(function (kind) {
      var present = active.includes(kind), badge = dotHost.querySelector('[data-status="' + kind + '"]');
      node.classList.toggle('ab-status-' + kind, present);
      if (!present) { if (badge) badge.remove(); return; }
      var matching = statuses.filter(function (effect) { return (String(effect.key || '') + ' ' + String(effect.name || '')).toLowerCase().includes(kind); });
      var rounds = Math.max.apply(null, matching.map(function (effect) { return finite(effect.remaining_rounds, 0); }));
      var text = (kind === 'poison' ? 'Poison' : 'Bleed') + (matching.length > 1 ? ' ×' + matching.length : '') + (rounds > 0 ? ' · ' + rounds + 'R' : '');
      if (!badge) { badge = el('span', '', 'ab-dot-badge ' + kind); badge.dataset.status = kind; dotHost.appendChild(badge); }
      badge.textContent = text; labels.push(text);
    });
    dotHost.hidden = !labels.length;
    if (labels.length) { node.title += ' · ' + labels.join(' · '); node.setAttribute('aria-label', node.getAttribute('aria-label') + ', ' + labels.join(', ')); }
  }
  function history(event, actors, targets) {
    var seq = sequence(Number(event.seq)); if (!seq || seq < observed) return; observed = seq;
    var source = actors.get(event.actor_id), name = source && source._combatUnit.name || 'Combatant';
    var action = String(event.ability_name || event.kind || 'Action'); byID('liveVisualCaption').textContent = name + ' · ' + action;
    var list = byID('liveVisualHistory');
    if (event.round && event.round !== lastRound) { list.appendChild(el('li', 'Round ' + event.round, 'round')); lastRound = event.round; }
    var parts = (targets || []).slice(0, 24).map(function (target) {
      var node = actors.get(target.target_id), unit = node && node._combatUnit || {hp_hidden: true}, hidden = health(unit).kind === 'concealed';
      var values = [], who = unit.name || 'Target';
      if (target.damage > 0 || target.damaged) values.push(hidden ? 'hit' : '−' + target.damage + ' HP');
      if (target.healing > 0 || target.healed) values.push(hidden ? 'healed' : '+' + target.healing + ' HP');
      if (target.absorbed > 0) values.push(hidden ? 'absorbed' : 'absorbed ' + target.absorbed);
      if (target.dodged) values.push('dodged'); if (target.blocked) values.push('blocked'); if (target.critical) values.push('critical');
      if (target.status) values.push(String(target.status).replace(/_/g, ' ')); if (target.defeated) values.push('defeated');
      return who + (values.length ? ': ' + values.join(', ') : '');
    });
    var summary = name + ' · ' + action + (parts.length ? ' → ' + parts.join('; ') : '');
    var row = list.querySelector('[data-event-seq="' + seq + '"]');
    if (!row) { row = el('li', summary); row.dataset.eventSeq = String(seq); list.appendChild(row); }
    else if (parts.length) row.textContent += (row.dataset.hasResults ? '; ' : ' → ') + parts.join('; ');
    if (parts.length) row.dataset.hasResults = 'true';
    while (list.children.length > 12) list.firstElementChild.remove();
    byID('liveAnimationAnnouncement').textContent = row.textContent;
  }
  function prepare(event, profile, actor, actors) {
    if (!host) return; history(event, actors);
    if (actor && profile.pose === 'cast') flourish(actor, 'seal', (profile.palette || [])[0], 300);
  }
  function hit(event, target, node, profile) {
    if (!node) return; var kind = impact(target);
    node.classList.add('ab-fight-target'); after(function () { node.classList.remove('ab-fight-target'); }, 260);
    flourish(node, kind, (profile.palette || [])[0]);
    var status = String(target.status || '').toLowerCase();
    if (event.kind === 'status') status += ' ' + String(event.ability_id || '') + ' ' + String(event.ability_name || '').toLowerCase();
    if (/poison/.test(status)) flourish(node, 'poison'); else if (/bleed/.test(status)) flourish(node, 'bleed'); else if (/stun/.test(status)) flourish(node, 'stun');
    if (target.defeated || node._presentationDefeated) return;
    var sprite = node.querySelector('.ab-actor-sprite'), base = 'scaleX(var(--ab-facing,1)) ';
    if (kind === 'dodge') move(sprite, [{transform: base + 'translateX(0)'}, {transform: base + 'translateX(-9px)', offset: .4}, {transform: base + 'translateX(0)'}], 210);
    else if (kind === 'block') move(sprite, [{transform: base + 'scaleY(1)'}, {transform: base + 'scaleY(.93)', offset: .35}, {transform: base + 'scaleY(1)'}], 180);
    else if (kind === 'critical' && global.liveCombatImpactEnabled !== false) move(sprite, [{transform: base + 'translateX(-5px)'}, {transform: base + 'translateX(3px)', offset: .35}, {transform: base + 'translateX(0)'}], 220);
    else if (kind === 'heal') move(sprite, [{transform: base + 'translateY(0)'}, {transform: base + 'translateY(-3px)', offset: .4}, {transform: base + 'translateY(0)'}], 260);
  }
  function playback(state, pending) {
    if (!host) return; var text = state === 'playing' ? pending + ' effect' + (pending === 1 ? '' : 's') + ' pending' : state === 'catchup' ? 'Caught up to live combat' : 'Live';
    byID('liveVisualPlayback').textContent = text; byID('liveAnimationSkip').disabled = state !== 'playing'; apply();
  }
  function fit() {
    if (!host) return;
    var height = host.clientHeight; if (!height) return;
    var compact = String(height < 190);
    if (host.dataset.visualCompact !== compact) host.dataset.visualCompact = compact;
    var caption = host.querySelector('.ab-fight-presentation'), depth = height < 190 ? 4 : 16;
    var top = Math.max(16, caption ? caption.offsetHeight + depth : 16);
    if (host.style.paddingTop !== top + 'px') host.style.paddingTop = top + 'px';
    if (host.style.paddingBottom !== '8px') host.style.paddingBottom = '8px';
    // Compact labels and padding can change layout. Measure after applying them,
    // then read every actor before changing any inherited sprite-size property.
    height = host.clientHeight;
    var sizes = [];
    host.querySelectorAll('.ab-pixel-party').forEach(function (party) {
      var units = Array.from(party.querySelectorAll('.ab-pixel-unit:not(.ab-departed)'));
      var rows = Math.max(1, ...units.map(function (unit) { return Number(unit.style.gridRow) || 1; }));
      var available = Math.max(0, (height - top - 12 - (rows - 1) * 7) / rows);
      units.forEach(function (unit) {
        var info = unit.querySelector('.ab-combat-unit-info');
        var size = Math.max(16, Math.min(110, available - info.offsetHeight - 6));
        sizes.push({unit: unit, value: size + 'px'});
      });
    });
    sizes.forEach(function (size) {
      if (size.unit.style.getPropertyValue('--fight-sprite-height') !== size.value) size.unit.style.setProperty('--fight-sprite-height', size.value);
    });
  }
  // Preparation names the action; reveal each authoritative target result at contact.
  api.contact = function (event, target, actors) { history(event, actors, [target]); };
  api.init = init; api.snapshot = snapshot; api.actor = actor; api.prepare = prepare; api.hit = hit; api.clear = clear; api.playback = playback; api.fit = fit;
  api.linger = function () { return prefs.linger; };
  api.effect = function (node, profile) { node.dataset.element = String(profile.element || 'physical'); };
  api.number = function (node, target, hidden) { if (!hidden) node.title = [target.damage > 0 ? 'Damage ' + target.damage : '', target.healing > 0 ? 'Healing ' + target.healing : '', target.absorbed > 0 ? 'Absorbed ' + target.absorbed : ''].filter(Boolean).join(' · '); };
})(window);
