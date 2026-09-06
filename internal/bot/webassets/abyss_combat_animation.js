/* Cosmetic playback only. The latest server snapshot always owns HP, targets,
   action availability and deadlines; animation callbacks never change them. */
(function () {
  'use strict';
  var session = '', cursor = 0, accepted = 0, queue = [], active = false;
  var generation = 0, timers = new Set(), animations = new Set(), actors = new Map();
  var latest = null, catchup = false, visible = true, initialized = false, batchDeadline = 0;
  var motion = window.matchMedia('(prefers-reduced-motion: reduce)');
  var speed = read('abyssAnimationSpeed', 'normal'), effects = read('abyssAnimationEffects', 'full');
  function read(key, fallback) { try { return localStorage.getItem(key) || fallback; } catch (_) { return fallback; } }
  function save(key, value) { try { localStorage.setItem(key, value); } catch (_) {} }
  function stage() { return document.getElementById('livePixelStage'); }
  function reduced() { return motion.matches || effects === 'reduced'; }
  function later(callback, delay) {
    var token = generation, timer = setTimeout(function () {
      timers.delete(timer); if (token === generation) callback();
    }, delay);
    timers.add(timer); return timer;
  }
  function animate(node, frames, duration) {
    if (reduced() || !node.animate) return;
    var animation = node.animate(frames, {duration: duration, easing: 'cubic-bezier(.2,.7,.3,1)'});
    animations.add(animation);
    animation.finished.then(function () { animations.delete(animation); }, function () { animations.delete(animation); });
  }
  function mark(state) {
    var host = stage(); if (!host) return;
    host.dataset.presentationState = state;
    host.dataset.lastEventSeq = String(cursor);
    host.dataset.effects = reduced() ? 'reduced' : 'full';
    host.dataset.playbackSpeed = speed;
  }
  function setFrame(node, pose, frame) {
    if (!node || !window.AbyssCombatArt) return;
    var unit = node._combatUnit, sprite = node.querySelector('.ab-actor-sprite');
    if (!unit || !sprite) return;
    var art = window.AbyssCombatArt.actorFrame(unit, pose, frame || 0);
    if (!art) return;
    var asset = art.asset + (window.__ASSET_VER__ ? '?v=' + encodeURIComponent(window.__ASSET_VER__) : '');
    sprite.classList.add('ab-posed-actor');
    sprite.style.backgroundImage = 'url("' + asset + '")';
    sprite.style.backgroundSize = art.size || (art.columns * 100) + '% ' + (art.rows * 100) + '%';
    sprite.style.backgroundPosition = art.position || (art.column * 100 / (art.columns - 1)) + '% ' + (art.row * 100 / (art.rows - 1)) + '%';
    sprite.dataset.pose = pose;
    sprite.dataset.rig = art.rig;
    node.dataset.pose = pose;
  }
  function rest(node) { setFrame(node, node._presentationDefeated || !node._pendingDefeat && node._combatUnit && node._combatUnit.hp === 0 && !node._combatUnit.hp_hidden ? 'defeat' : 'idle', 0); }
  function clear() {
    generation++; timers.forEach(clearTimeout); timers.clear();
    animations.forEach(function (animation) { animation.cancel(); }); animations.clear();
    queue = []; active = false;
    var host = stage();
    if (host) host.querySelectorAll('.ab-combat-effect,.ab-combat-number,.ab-combat-announcement').forEach(function (node) { node.remove(); });
    actors.forEach(function (node) {
      node.classList.remove('ab-performing', 'ab-reacting'); node._pendingDefeat = false;
      var unit = node._combatUnit || {};
      node._presentationDefeated = !node._departed && !unit.hp_hidden && unit.hp === 0;
      node.classList.toggle('ab-defeated', node._presentationDefeated);
      node.classList.toggle('ab-departed', !!node._departed);
      if (node._departed) { node.disabled = true; node.setAttribute('aria-hidden', 'true'); }
      rest(node);
    });
  }
  function reset(id) {
    if (session === (id || '')) return;
    clear(); session = id || ''; cursor = 0; accepted = 0; latest = null; catchup = false; actors.clear();
    var announcement = document.getElementById('liveAnimationAnnouncement'); if (announcement) announcement.textContent = '';
    mark('idle');
  }
  function init() {
    if (initialized || !stage()) return;
    initialized = true;
    var speedSelect = document.getElementById('liveAnimationSpeed'), effectsSelect = document.getElementById('liveAnimationEffects');
    speed = speed === 'fast' ? 'fast' : 'normal'; effects = effects === 'reduced' ? 'reduced' : 'full';
    if (speedSelect) { speedSelect.value = speed; speedSelect.onchange = function () { speed = this.value; save('abyssAnimationSpeed', speed); mark(active ? 'playing' : 'idle'); }; }
    if (effectsSelect) { effectsSelect.value = effects; effectsSelect.onchange = function () { effects = this.value; save('abyssAnimationEffects', effects); preferencesChanged(); }; }
    if (motion.addEventListener) motion.addEventListener('change', preferencesChanged);
    document.addEventListener('visibilitychange', function () {
      if (document.hidden) { clear(); catchup = true; cursor = accepted; mark('catchup'); }
    });
    var compact = window.innerWidth <= 760;
    window.addEventListener('resize', function () {
      var nextCompact = window.innerWidth <= 760;
      if (nextCompact === compact || !latest || !window.renderAbyssEventStage) return;
      compact = nextCompact; window.renderAbyssEventStage(latest);
    });
    if (window.IntersectionObserver) new IntersectionObserver(function (entries) {
      visible = entries[0].isIntersecting;
    }).observe(stage());
    // Two authored idle poses; no DOM work for hidden, offscreen or still scenes.
    var idleFrame = 0;
    setInterval(function () {
      if (!session || document.hidden || !visible || reduced()) return;
      idleFrame = 1 - idleFrame;
      actors.forEach(function (node) { if (node.dataset.pose === 'idle' && !node.classList.contains('ab-departed')) setFrame(node, 'idle', idleFrame); });
    }, 750);
  }
  function preferencesChanged() {
    if (reduced()) animations.forEach(function (animation) { animation.cancel(); });
    mark(active ? 'playing' : 'idle');
  }
  function register(node, unit) {
    var id = String(unit.entity_id || unit.id);
    node._combatUnit = unit; actors.set(id, node);
    if (window.AbyssCombatArt) node.style.setProperty('--actor-accent', 'url("' + window.AbyssCombatArt.actorProfile(unit).accent + '")');
    if (!node.dataset.pose || !active) rest(node);
  }
  function point(node) {
    var host = stage().getBoundingClientRect();
    var sprite = node && node.querySelector('.ab-actor-sprite');
    var rect = (sprite || node || stage()).getBoundingClientRect();
    return {x: rect.left - host.left + rect.width / 2, y: rect.top - host.top + rect.height * .55};
  }
  function label(text, event, className) {
    var node = document.createElement('span'); node.className = 'ab-combat-announcement ' + (className || '');
    node.dataset.eventSeq = String(event.seq || 0); node.textContent = text;
    var announcement = document.getElementById('liveAnimationAnnouncement');
    if (announcement) announcement.textContent = text;
    stage().appendChild(node); later(function () { node.remove(); }, 850);
  }
  function effect(event, target, profile, phase, duration, sourceID) {
    var source = actors.get(sourceID || event.actor_id), destination = actors.get(target.target_id);
    if (!destination) return;
    var from = point(source || destination), to = point(destination), node = document.createElement('span');
    node.className = 'ab-combat-effect ab-effect-' + phase;
    node.dataset.eventSeq = String(event.seq); node.dataset.actorId = event.actor_id || ''; node.dataset.targetId = target.target_id;
    node.dataset.abilityId = event.ability_id || event.kind; node.dataset.effectFamily = profile.family || 'physical';
    node.setAttribute('aria-hidden', 'true');
    node.style.setProperty('--effect-color', (profile.palette || ['#f2bd5b'])[0]);
    if (window.AbyssCombatArt) node.style.backgroundImage = 'url("' + window.AbyssCombatArt.effectFrame(profile, phase, 0) + '")';
    node.style.left = to.x + 'px'; node.style.top = to.y + 'px';
    if (phase === 'prepare') { node.style.left = from.x + 'px'; node.style.top = from.y + 'px'; }
    if (phase === 'travel' && !reduced()) {
      node.style.left = from.x + 'px'; node.style.top = from.y + 'px';
      var dx = to.x - from.x, dy = to.y - from.y, angle = Math.atan2(dy, dx) * 180 / Math.PI;
      if (/^(lightning|beam|ray|drain|breath)$/.test(profile.family)) {
        var length = Math.hypot(dx, dy);
        node.style.width = length + 'px'; node.style.height = '44px';
        node.style.backgroundSize = '100% 100%';
        node.style.transform = 'translateY(-50%) rotate(' + angle + 'deg)'; node.style.transformOrigin = '0 50%';
        if (profile.family === 'lightning') {
          var path = 'M0 22';
          for (var step = 1; step < 12; step++) path += ' L' + (step * 10) + ' ' + (step % 2 ? 7 : 37);
          path += ' L120 22';
          node.style.backgroundImage = 'url("data:image/svg+xml,' + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 44" preserveAspectRatio="none"><path d="' + path + '" fill="none" stroke="' + profile.palette[0] + '" stroke-width="4"/></svg>') + '")';
          node.style.backgroundSize = '100% 100%';
        }
        animate(node, [{opacity: 0}, {opacity: 1, offset: .18}, {opacity: .65, offset: .65}, {opacity: 0}], duration);
      } else {
        var arc = /^(arrow|volley|barrage|wave)$/.test(profile.family) ? -28 - (profile.variant % 3) * 8 : 0;
        animate(node, [{transform: 'translate(-50%,-50%) rotate(' + angle + 'deg)'}, {offset: .5, transform: 'translate(calc(-50% + ' + dx / 2 + 'px),calc(-50% + ' + (dy / 2 + arc) + 'px)) rotate(' + angle + 'deg)'}, {transform: 'translate(calc(-50% + ' + dx + 'px),calc(-50% + ' + dy + 'px)) rotate(' + angle + 'deg)'}], duration);
      }
    } else animate(node, [{opacity: .3, transform: 'translate(-50%,-50%) scale(.55)'}, {opacity: 1, offset: .25, transform: 'translate(-50%,-50%) scale(1.1)'}, {opacity: 0, transform: 'translate(-50%,-50%) scale(1.2)'}], duration);
    stage().appendChild(node);
    if (!reduced() && !(phase === 'travel' && profile.family === 'lightning')) later(function () { if (window.AbyssCombatArt) node.style.backgroundImage = 'url("' + window.AbyssCombatArt.effectFrame(profile, phase, 1) + '")'; }, duration / 2);
    later(function () { node.remove(); }, duration);
  }
  function outcome(event, target) {
    var actor = actors.get(target.target_id); if (!actor) return;
    if (target.status === 'revived' || target.healing > 0 && actor._presentationDefeated && actor._combatUnit.hp > 0) {
      actor._presentationDefeated = false; actor._pendingDefeat = false; actor.classList.remove('ab-defeated');
      var unit = actor._combatUnit;
      actor.disabled = !!actor._departed || !unit.hp_hidden && unit.hp === 0 || !unit.is_player && !actor.classList.contains('hostile');
      rest(actor);
    }
    var hidden = actor._combatUnit && actor._combatUnit.hp_hidden, parts = [];
    if (!hidden && target.damage > 0) parts.push({text: (target.critical ? 'CRIT ' : '') + '−' + window.fmtNum(target.damage), kind: target.critical ? 'critical' : 'damage'});
    if (!hidden && target.healing > 0) parts.push({text: '+' + window.fmtNum(target.healing), kind: 'healing'});
    if (!hidden && target.absorbed > 0) parts.push({text: 'ABSORB ' + window.fmtNum(target.absorbed), kind: 'absorb'});
    if (target.blocked) parts.push({text: 'BLOCK', kind: 'block'});
    if (target.dodged) parts.push({text: 'DODGE', kind: 'dodge'});
    if (target.status) parts.push({text: String(target.status).replace(/_/g, ' ').toUpperCase(), kind: 'status'});
    if (target.defeated) parts.push({text: 'DEFEATED', kind: 'defeat'});
    if (!parts.length && hidden && (target.healed || target.healing > 0)) parts.push({text: 'HEAL', kind: 'healing'});
    else if (!parts.length && hidden && (target.damaged || target.damage > 0)) parts.push({text: 'HIT', kind: 'damage'});
    var anchor = point(actor), host = stage(), width = host.clientWidth;
    parts.forEach(function (part) {
      var previous = Array.from(host.querySelectorAll('.ab-combat-number')).filter(function (n) { return n.dataset.targetId === target.target_id; });
      // Keep a readable four-line stack per target during compressed bursts.
      if (previous.length >= 4) { previous.shift().remove(); }
      var occupied = new Set(previous.map(function (n) { return Number(n.dataset.lane); }));
      var lane = 0; while (occupied.has(lane)) lane++;
      var node = document.createElement('span'); node.className = 'ab-combat-number ' + part.kind;
      node.dataset.eventSeq = String(event.seq); node.dataset.targetId = target.target_id; node.dataset.lane = String(lane);
      node.textContent = part.text;
      node.style.left = Math.max(66, Math.min(width - 66, anchor.x)) + 'px';
      node.style.top = (Math.max(95, anchor.y - 25) - lane * 22) + 'px';
      host.appendChild(node);
      animate(node, [{opacity: 0, transform: 'translate(-50%,6px)'}, {opacity: 1, offset: .12, transform: 'translate(-50%,0)'}, {opacity: 1, offset: .8, transform: 'translate(-50%,-7px)'}, {opacity: 0, transform: 'translate(-50%,-13px)'}], 760);
      later(function () { node.remove(); }, 800);
    });
    if (!reduced() && (target.damage > 0 || target.blocked || target.absorbed > 0)) {
      actor._poseEvent = event.seq;
      setFrame(actor, 'hurt'); actor.classList.add('ab-reacting');
      animate(actor.querySelector('.ab-actor-sprite'), [{filter: 'brightness(1.6)'}, {filter: 'brightness(1)'}], 170);
      later(function () { if (actor._poseEvent !== event.seq) return; actor.classList.remove('ab-reacting'); rest(actor); }, 180);
    }
    if (target.defeated) { actor._pendingDefeat = false; actor._presentationDefeated = true; actor.disabled = true; setFrame(actor, 'defeat'); actor.classList.add('ab-defeated'); actor.classList.remove('ab-departed'); actor.removeAttribute('aria-hidden'); }
  }
  function playNext() {
    if (!queue.length) {
      active = false;
      actors.forEach(function (node) { rest(node); if (node._departed && !node._presentationDefeated) { node.classList.add('ab-departed'); node.setAttribute('aria-hidden', 'true'); } });
      mark('idle'); return;
    }
    active = true; mark('playing');
    var event = queue.shift(), actor = actors.get(event.actor_id);
    var unit = actor && actor._combatUnit || {};
    var profile = window.AbyssCombatArt ? window.AbyssCombatArt.profileFor(Object.assign({}, event, {weapon_type: unit.weapon_type, weapon_name: unit.weapon_name})) : {family: event.element || 'physical', palette: ['#f2bd5b']};
    var fast = speed === 'fast' || (latest && latest.pause_mode === 'fast');
    var terminal = latest && (latest.phase === 'complete' || latest.phase === 'failed');
    var duration = Math.max(12, Math.min(fast ? 190 : 480, (batchDeadline - performance.now()) / Math.max(1, queue.length + 1)));
    if (reduced()) duration = Math.min(duration, 100);
    var targets = (event.targets || []).slice(0, 24), pose = profile.pose || (event.kind === 'attack' || event.kind === 'pet' ? 'attack' : 'cast');
    if (actor) { actor._poseEvent = event.seq; actor.classList.add('ab-performing'); setFrame(actor, pose, 0); }
    if (event.kind === 'phase') label(event.ability_name || 'PHASE CHANGE', event, 'boss');
    else if (event.kind === 'ultimate') label(event.ability_name || 'ULTIMATE', event, 'ultimate');
    else if (actor) {
      var name = actor.querySelector('.ab-combat-action-name');
      if (name) name.textContent = event.ability_name || event.kind;
    }
    later(function () {
      if (actor) setFrame(actor, pose, 1);
      if (actor && pose === 'attack') animate(actor.querySelector('.ab-actor-sprite'), [{transform: 'scaleX(var(--ab-facing,1)) translateX(0)'}, {transform: 'scaleX(var(--ab-facing,1)) translateX(12px)', offset: .45}, {transform: 'scaleX(var(--ab-facing,1)) translateX(0)'}], duration * .5);
      targets.forEach(function (target, index) {
        if (profile.family === 'lightning' && targets.length > 1) {
          var linkDuration = duration * .3 / targets.length;
          later(function () { effect(event, target, profile, 'travel', linkDuration, index ? targets[index - 1].target_id : event.actor_id); }, index * linkDuration);
        } else effect(event, target, profile, profile.projectile || profile.family === 'lightning' ? 'travel' : 'prepare', duration * .3);
      });
    }, duration * .25);
    later(function () {
      targets.forEach(function (target) { effect(event, target, profile, 'impact', Math.max(140, duration * .4)); outcome(event, target); });
      var healing = targets.some(function (target) { return target.healing > 0; });
      var defeated = targets.some(function (target) { return target.defeated; });
      var cue = defeated ? 'defeat' : healing ? 'heal' : event.kind === 'ultimate' ? 'ultimate' : pose === 'cast' ? 'cast' : 'hit';
      if (window.playLiveCombatCue) window.playLiveCombatCue(cue, event.kind === 'ultimate' ? .9 : .4);
      if (!reduced() && window.pulseLiveCombatStage && (defeated || event.kind === 'ultimate')) window.pulseLiveCombatStage(cue);
    }, duration * .56);
    later(function () {
      if (actor) { actor.classList.remove('ab-performing'); rest(actor); }
      cursor = Math.max(cursor, Number(event.seq) || 0); mark('playing'); playNext();
    }, duration);
  }
  function ingest(state) {
    init(); reset(state.session_id);
    var first = !latest, previousVersion = latest && latest.version;
    latest = state;
    var events = Array.isArray(state.presentation_events) ? state.presentation_events : [];
    var nextCursor = Math.max(Number(state.presentation_cursor) || 0, ...events.map(function (event) { return Number(event.seq) || 0; }));
    if (first || catchup || document.hidden) {
      clear(); accepted = cursor = nextCursor; catchup = document.hidden;
      mark(first ? 'idle' : 'catchup');
      if (first) actors.forEach(function (node) { if (node._combatUnit.role === 'boss') { node.classList.add('ab-boss-arrival'); later(function () { node.classList.remove('ab-boss-arrival'); }, 650); } });
      return;
    }
    if (Number(state.version) < Number(previousVersion)) return;
    var sequences = new Set();
    var fresh = events.filter(function (event) { var seq = Number(event.seq); if (seq <= accepted || sequences.has(seq)) return false; sequences.add(seq); return true; }).sort(function (a, b) { return a.seq - b.seq; });
    if (!fresh.length) return;
    if (fresh[0].seq > accepted + 1 || queue.length + fresh.length > 48) {
      clear(); accepted = cursor = nextCursor; mark('catchup'); return;
    }
    accepted = Math.max(accepted, nextCursor); queue.push.apply(queue, fresh);
    var budget = state.phase === 'complete' || state.phase === 'failed' ? 620 : speed === 'fast' || state.pause_mode === 'fast' ? 900 : 1600;
    batchDeadline = active ? Math.min(batchDeadline, performance.now() + budget) : performance.now() + budget;
    if (!active) playNext();
  }
  function connection(state) {
    if (state === 'reconnecting' || state === 'connecting' || state === 'polling') {
      if (latest) { clear(); catchup = true; cursor = accepted; mark('catchup'); }
    }
  }
  window.AbyssCombatAnimation = {isPlaying: function (id) { return session === id && active; }, reset: reset, register: register, ingest: ingest, connection: connection, dispose: function () { reset(''); }};
})();

// Keep actor buttons and formation slots stable while server routing IDs change.
window.renderAbyssEventStage = function (state) {
  resetLivePixelState(state.session_id);
  var stage = document.getElementById('livePixelStage');
  stage.classList.add('ab-event-presentation');
  if (window.AbyssCombatArt) {
    var biome = document.getElementById('biomeChip'), depth = document.getElementById('depthNum');
    var scenery = window.AbyssCombatArt.backdrop({biome: state.biome || biome && biome.textContent, depth: state.depth || Number(depth && depth.textContent)});
    if (stage.dataset.biomeArt !== scenery.key) { stage.dataset.biomeArt = scenery.key; stage.style.setProperty('--combat-scenery', 'url("' + scenery.image + '")'); }
  }
  function renderSide(hostID, units, hostile) {
    var host = document.getElementById(hostID), slots = host._formationSlots || (host._formationSlots = new Map());
    var present = new Set();
    // Choose capacity once; departures leave a vacant slot instead of moving survivors.
    var layout = window.innerWidth <= 760 ? 'compact' : 'wide';
    if (!host.dataset.formationColumns || !slots.size || host.dataset.formationLayout !== layout) host.dataset.formationColumns = String(Math.max(1, Math.min(layout === 'compact' ? 3 : 6, Math.max(slots.size, (units || []).length))));
    host.dataset.formationLayout = layout;
    var columns = Number(host.dataset.formationColumns);
    host.style.setProperty('--formation-columns', columns);
    host.classList.toggle('crowded', Math.max(slots.size, (units || []).length) > 4);
    (units || []).forEach(function (unit, index) {
      var entity = String(unit.entity_id || unit.id); present.add(entity);
      var button = Array.from(stage.querySelectorAll('.ab-pixel-unit')).find(function (node) { return node.dataset.entityId === entity; });
      var created = !button;
      if (created) { button = document.createElement('button'); button.type = 'button'; button.className = 'ab-pixel-unit'; button.dataset.entityId = entity; }
      button._departed = false;
      if (!unit.hp_hidden && unit.hp > 0) button._presentationDefeated = false;
      button._pendingDefeat = (state.presentation_events || []).some(function (event) { return event.seq > Number(stage.dataset.lastEventSeq || 0) && (event.targets || []).some(function (target) { return target.target_id === entity && target.defeated; }); });
      if (!slots.has(entity)) slots.set(entity, slots.size);
      var slot = slots.get(entity);
      button.style.gridColumn = String(hostile ? (slot % columns) + 1 : columns - (slot % columns));
      button.style.gridRow = String(Math.floor(slot / columns) + 1);
      button.dataset.formationSlot = String(slot);
      button.dataset.formationRow = /back|ranger|caster|support|mage/.test(String(unit.position || '') + ' ' + String(unit.role || '')) ? 'backline' : 'frontline';
      var artKey = String(unit.art_key || ((hostile ? 'monster:' : 'ally:') + String(unit.name || unit.id)));
      var art = hostile || !unit.is_player ? liveEnemyArt(unit) : null, hpHidden = hostile && !!unit.hp_hidden;
      var boss = hostile && String(unit.role || '').toLowerCase() === 'boss';
      var elite = hostile && !boss && (/^nemesis:|behemoth|gatekeeper|lich|knight|invader/i.test(unit.name || ''));
      var classes = {hostile: hostile, 'ally-converted': !hostile && !unit.is_player, 'boss-tier': boss, 'elite-tier': elite, 'hp-concealed': hpHidden, shielded: !hostile && unit.shield > 0, 'weakness-ready': !!unit.weakness_ready, selected: liveSelectedTarget === unit.id, 'ab-departed': false, 'ab-defeated': button._presentationDefeated || !button._pendingDefeat && !hpHidden && unit.hp === 0};
      Object.keys(classes).forEach(function (name) { button.classList.toggle(name, classes[name]); });
      button.removeAttribute('aria-hidden');
      button.classList.remove('affinity-fire', 'affinity-frost', 'affinity-toxic', 'affinity-void', 'affinity-metal', 'affinity-giant');
      var affinity = hostile ? liveEnemyAffinity(unit).trim() : ''; if (affinity) button.classList.add(affinity);
      button.disabled = (!hpHidden && unit.hp === 0) || (!hostile && !unit.is_player);
      button.dataset.target = unit.id;
      button.setAttribute('aria-label', (hostile ? 'Target ' : 'Inspect ') + unit.name + ', ' + (hpHidden ? 'health concealed' : livePct(unit.hp, unit.max_hp) + ' percent health' + liveShieldAria(unit)) + liveWeaknessAria(unit) + liveAffixAria(unit));
      button.setAttribute('aria-pressed', liveSelectedTarget === unit.id ? 'true' : 'false');
      var effects = (unit.effects || []).slice(0, 4).map(function (effect) { return liveEffectChip(effect, true); }).join('');
      var family = art ? (art.family || art.atlas) : 'player', cell = art ? art.cell : livePlayerCell(unit);
      var spriteClass = 'ab-actor-sprite' + (art && art.atlas === 'catalog' ? ' ab-catalog-actor' : '');
      var spriteStyle = art && art.atlas === 'catalog' ? liveUniqueArtStyle(artKey, art.family) : 'background-position:' + liveSpritePosition(cell);
      var identity = liveArtIdentity(artKey);
      if (created) button.innerHTML = '<span class="ab-combat-unit-info"></span><span class="' + spriteClass + '" data-art-signature="' + identity.signature + '" data-art-sheet="' + family + '" style="' + spriteStyle + '" aria-hidden="true"></span><span class="ab-pixel-shadow" aria-hidden="true"></span><span class="ab-pixel-effects"></span><span class="ab-combat-action-name" aria-hidden="true"></span>';
      var portrait = art ? '<span class="ab-combat-portrait ab-pixel-icon ab-catalog-icon" style="' + liveUniqueArtStyle(artKey, art.family) + '" aria-hidden="true"></span>' : '';
      var info = '<span class="ab-pixel-name">' + portrait + '<b>' + (unit.revenge ? '◎ ' : '') + consEsc(unit.name) + '</b><span>' + consEsc(unit.revenge ? 'REVENGE TARGET' : unit.role || unit.element || '') + '</span></span>' + (hostile ? liveWeaknessWindowMark(unit, 'pixel') : '') + (!hostile ? liveShieldBar(unit, 'overhead') : '') + (hpHidden ? '<span class="ab-overhead-hp concealed"><i></i><em>??</em></span>' : '<span class="ab-overhead-hp"><i style="width:' + livePct(unit.hp, unit.max_hp) + '%"></i><em>' + livePct(unit.hp, unit.max_hp) + '%</em></span>') + (boss || elite ? '<span class="ab-pixel-rank">' + (boss ? 'BOSS' : 'ELITE') + '</span>' : '');
      var infoNode = button.querySelector('.ab-combat-unit-info'); if (infoNode.innerHTML !== info) infoNode.innerHTML = info;
      var effectsNode = button.querySelector('.ab-pixel-effects'); if (effectsNode.innerHTML !== effects) effectsNode.innerHTML = effects;
      var intent = (state.enemy_intents || []).find(function (value) { return value.enemy_id === unit.id; });
      button.classList.toggle('ab-danger-intent', !!intent && /heavy|ultimate|special|cast|charge|heal|aoe/.test(intent.kind || ''));
      button.dataset.intent = intent ? String(intent.ability || intent.kind || '') : '';
      button.onclick = function () { selectLiveTarget(button.dataset.target); };
      if (button.parentNode !== host) host.appendChild(button);
      window.AbyssCombatAnimation.register(button, unit);
    });
    Array.from(host.children).forEach(function (node) {
      if (node.dataset.entityId && !present.has(node.dataset.entityId)) {
        var pending = (state.presentation_events || []).some(function (event) { return event.seq > Number(stage.dataset.lastEventSeq || 0) && (event.actor_id === node.dataset.entityId || (event.targets || []).some(function (target) { return target.target_id === node.dataset.entityId; })); });
        node._departed = true; node.disabled = true;
        node.classList.toggle('ab-departed', !pending && !node._presentationDefeated);
        if (!pending && !node._presentationDefeated) node.setAttribute('aria-hidden', 'true'); else node.removeAttribute('aria-hidden');
      }
    });
  }
  renderSide('livePixelAllies', state.allies, false); renderSide('livePixelEnemies', state.enemies, true);
  window.AbyssCombatAnimation.ingest(state);
  livePixelPrevious.version = Number(state.version) || 0;
};
