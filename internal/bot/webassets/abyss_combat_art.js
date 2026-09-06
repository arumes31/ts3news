/* Abyss combat artwork: authored pose atlases + deterministic semantic spell geometry.
 * No game state is inferred here. Effects are requested only for confirmed events.
 * Existing catalog portraits remain the authoritative character/item identities. */
(function (global) {
  'use strict';
  var palettes = {
    physical: ['#f4d293', '#e89352', '#fff3cc'], fire: ['#ff783c', '#d63d46', '#ffe59c'],
    frost: ['#83dcff', '#5c8ce3', '#edfbff'], storm: ['#c6b2ff', '#7798ff', '#f6edff'],
    nature: ['#91dc93', '#409e91', '#e1ffae'], toxic: ['#b4db55', '#787bd6', '#e6fca5'],
    water: ['#76cddc', '#487eaf', '#d9f7ee'], air: ['#b8ebdc', '#74acae', '#efffeb'],
    holy: ['#f2d78a', '#ddab71', '#fff4cf'],
    shadow: ['#b49be2', '#6253a3', '#ddd0ff'], void: ['#dd90dd', '#7854ad', '#f1cbff'],
    blood: ['#e88390', '#9d4567', '#ffd1ca'], arcane: ['#a5b8ff', '#7372bd', '#e0e6ff']
  };
  var poses = { idle: [0, 1], attack: [2, 3], cast: [4, 5], hurt: [6], defeat: [7] };
  var rigs = ['knight', 'ranger', 'wizard', 'druid', 'rogue', 'wolf', 'lich', 'dragon',
    'slime', 'spider', 'goblin', 'golem', 'demon', 'ghost', 'serpent', 'orc',
    'rat', 'bat', 'zombie', 'skeleton', 'troll', 'kraken', 'void-lord', 'chronos',
    'gorgoroth', 'malakor', 'azazoth', 'abyssus', 'scribe', 'mnemos', 'remembers', 'gatekeeper'];
  var atlasAssets = ['/static/abyss_combat_roles_v2.png', '/static/abyss_combat_creatures_v2.png',
    '/static/abyss_combat_bestiary_v2.png', '/static/abyss_combat_bosses_v2.png'];
  // Measured transparent row separators in the generated 1254px originals.
  // The artwork is preserved byte-for-byte; proportional crops account for its uneven row spacing.
  var atlasRows = [[0, 158, 318, 476, 638, 783, 924, 1086, 1254],
    [0, 144, 298, 441, 617, 789, 947, 1076, 1254],
    [0, 143, 281, 435, 591, 758, 900, 1056, 1254],
    [0, 155, 312, 466, 625, 786, 941, 1085, 1254]];
  var effects = new Map();
  var profiles = new Map();
  var actorProfiles = new Map();
  var backdropCache = new Map();
  var verbs = ['annihilating', 'devastating', 'obliterating', 'shattering', 'eradicating',
    'decimating', 'destroying', 'crushing', 'smashing', 'pulverizing', 'vaporizing',
    'incinerating', 'freezing', 'electrifying', 'corrupting', 'purifying', 'banishing',
    'summoning', 'channeling', 'unleashing', 'igniting', 'extinguishing', 'rending',
    'cleaving', 'piercing', 'shredding', 'blasting', 'bombarding', 'storming', 'ravaging',
    'consuming', 'devouring', 'absorbing', 'reflecting', 'amplifying', 'nullifying',
    'silencing', 'blinding', 'stunning', 'paralyzing', 'poisoning', 'cursing', 'blessing',
    'healing', 'shielding', 'empowering', 'enraging', 'terrifying', 'inspiring', 'transcending'];
  var builtins = {
    basic_attack: ['Strike', 'strike', 'physical'], attack: ['Strike', 'strike', 'physical'],
    defend: ['Guard', 'shield', 'physical'], pet_attack: ['Companion Strike', 'fang', 'physical'],
    pet_pounce: ['Companion Pounce', 'fang', 'physical'], pet_healing_spell: ['Healing Spell', 'heal', 'nature'],
    pet_mending_cry: ['Mending Cry', 'heal', 'holy'], betrayal: ['Betrayal', 'slash', 'shadow'],
    support_strike: ['Support Strike', 'strike', 'holy'], lifesteal: ['Lifesteal', 'drain', 'blood'],
    bloodlust: ['Bloodlust', 'enrage', 'blood'], chain_attack: ['Chain Attack', 'lightning', 'storm'],
    overkill_cleave: ['Overkill Cleave', 'slash', 'physical'], thorns: ['Thorns', 'reflect', 'nature'],
    parry: ['Parry', 'reflect', 'physical'], poison: ['Poison', 'poison', 'toxic'],
    regeneration: ['Regeneration', 'heal', 'nature'], curse: ['Curse', 'curse', 'shadow'],
    corruption_backlash: ['Corruption Backlash', 'burst', 'void'], zone_hazard: ['Zone Hazard', 'quake', 'physical'],
    storm: ['Storm', 'storm', 'storm'], fatigue: ['Fatigue', 'fatigue', 'shadow'],
    mind_control: ['Mind Control', 'mark', 'void'], death_explosion: ['Death Explosion', 'nova', 'fire'],
    death_curse: ['Death Curse', 'curse', 'shadow'], death_summon: ['Death Summon', 'summon', 'void'],
    phoenix: ['Phoenix', 'revive', 'fire'], enrage_warning: ['Enrage Warning', 'warning', 'blood'],
    enrage: ['Enrage', 'enrage', 'blood'], summon: ['Summon', 'summon', 'void'],
    interrupted: ['Interrupted', 'silence', 'arcane'], summon_arrival: ['Summon Arrival', 'summon', 'void'],
    fire_phase: ['Fire Phase', 'flare', 'fire'], void_barrier: ['Void Barrier', 'shield', 'void'],
    hypnotic_pulse: ['Hypnotic Pulse', 'mark', 'void'], sleep_shockwave: ['Sleep Shockwave', 'wave', 'shadow']
  };

  function hash(text) {
    var value = 2166136261;
    String(text).split('').forEach(function (letter) { value = Math.imul(value ^ letter.charCodeAt(0), 16777619); });
    return value >>> 0;
  }
  function boundedCache(cache, key, value, limit) {
    if (cache.size >= limit) cache.delete(cache.keys().next().value);
    cache.set(key, value);
    return value;
  }
  function catalog() { return global.AB_COMBAT_CATALOG || global.AB_EXACT_ICON_MANIFEST || {}; }
  function canonicalKey(key) { return String(key).replace(/^(item:artifact:\d+):.*$/, '$1'); }
  function lookup(key) { key = canonicalKey(key); return catalog()[key] || (global.AB_EXACT_ICON_MANIFEST || {})[key]; }
  function keyFor(option) {
    var kind = String(option.kind || 'skill').toLowerCase();
    var id = String(option.ability_id || option.id || option.name || option.ability_name || 'attack');
    if (option.art_key) return canonicalKey(option.art_key);
    if (kind === 'attack' && (!option.ability_id && !option.id || /^(basic_attack|attack)$/.test(id))) return 'attack:basic_attack';
    if (kind === 'defend') return 'defend:defend';
    if (catalog()[id]) return id;
    if (/^(item|relic|artifact|potion)$/.test(kind)) kind = 'item';
    return canonicalKey(kind + ':' + id);
  }
  function elementFor(name, element) {
    var text = String(element || '').toLowerCase();
    if (palettes[text]) return text;
    if (text === 'earth') return 'nature';
    text += ' ' + String(name || '').toLowerCase();
    if (/fire|fiery|flam|cinder|pyre|inciner|ignit|burn/.test(text)) return 'fire';
    if (/frost|icy|ice|freez|blizzard|snow/.test(text)) return 'frost';
    if (/lightning|storm|thunder|electr|shock/.test(text)) return 'storm';
    if (/\bair\b|wind|gale|zephyr/.test(text)) return 'air';
    if (/poison|toxic|venom|blight|acid|corrupt/.test(text)) return 'toxic';
    if (/void|oblivion|starless|abyssal|annihilat/.test(text)) return 'void';
    if (/shadow|dark|curse|necrot|death|night|terrify/.test(text)) return 'shadow';
    if (/blood|vamp|leech|rage|ravag/.test(text)) return 'blood';
    if (/holy|light|divin|radiant|bless|puri|soul|spirit|heal/.test(text)) return 'holy';
    if (/nature|earth|root|leaf|moss|druid|mend|vine/.test(text)) return 'nature';
    if (/water|fog|tide|rain|drown/.test(text)) return 'water';
    if (/arcane|magic|mana|spark|transcend|summon|channel/.test(text)) return 'arcane';
    return 'physical';
  }
  // Engine elements group frost into Water and lightning into Air, and many
  // named skills deal Physical damage. Preserve that state; select artwork
  // from the authored name when it explicitly describes a visual theme.
  function visualElement(name, element) {
    var theme = elementFor(name, '');
    return theme === 'physical' ? elementFor('', element) : theme;
  }
  function familyFor(name, kind, element) {
    var text = String(name || '').toLowerCase();
    if (kind === 'defend') return 'shield';
    if (kind === 'relic') return 'restore';
    if (kind === 'companion' && /free-for-all/.test(text)) return 'fang';
    if (/reviv|resurrect|phoenix/.test(text)) return 'revive';
    if (kind === 'item' && /repair/.test(text)) return 'repair';
    if (kind === 'heal' || /\b(heal|mend|revival|rejuvenation|restore|restoration)\b/.test(text)) return 'heal';
    if (kind === 'item' && /potion|elixir|tonic|draught/.test(text)) return 'potion';
    if (/shield|ward|barrier|aegis|guard/.test(text)) return 'shield';
    if (/chain lightning|lightning|thunder/.test(text)) return 'lightning';
    if (/arrow|shot|snipe/.test(text)) return 'arrow';
    if (/drain|leech|siphon/.test(text)) return 'drain';
    if (/curse|hex|silence|terror/.test(text)) return 'curse';
    if (/quake|sunder|fissure/.test(text)) return 'quake';
    if (/focus|mark|target/.test(text)) return 'mark';
    if (/summon|companion|pet/.test(kind + ' ' + text)) return 'summon';
    var nouns = ['onslaught', 'barrage', 'volley', 'thrust', 'slash', 'bolt', 'beam', 'ray',
      'pulse', 'surge', 'flare', 'burst', 'nova', 'rage', 'wrath', 'fury', 'storm', 'wave', 'blast', 'strike'];
    for (var i = 0; i < nouns.length; i++) {
      if (new RegExp('\\b' + nouns[i] + '\\b').test(text)) return nouns[i];
    }
    if (/bash|smash|punch|slam/.test(text)) return 'strike';
    if (/bite|fang/.test(text)) return 'fang';
    if (/claw|rend|cleav/.test(text)) return 'slash';
    if (/breath/.test(text)) return 'breath';
    if (kind === 'attack') return 'strike';
    return element === 'physical' ? 'strike' : 'bolt';
  }
  function profileFor(option) {
    option = option || {};
    var key = keyFor(option), exact = lookup(key), builtin = builtins[String(option.ability_id || option.id || key.split(':').pop())];
    var cacheKey = [key, option.kind, option.element, option.name || option.ability_name, option.weapon_type, option.weapon_name].join('|');
    if (profiles.has(cacheKey)) return profiles.get(cacheKey);
    var name = (exact && exact.name) || option.name || option.ability_name || (builtin && builtin[0]) || key;
    var signatureMatch=String(option.ability_id||option.id||key.split(':').pop()).match(/^CLASS_([a-z]+)_(build|finish)$/);
    if(signatureMatch){
      var signatureStyles={vanguard:['shield','strike','holy'],berserker:['slash','rage','blood'],marksman:['mark','arrow','frost'],beastmaster:['mark','summon','nature'],elementalist:['flare','nova','fire'],chronomancer:['bolt','pulse','arcane'],oracle:['heal','flare','holy'],geomancer:['shield','quake','nature'],bloodblade:['drain','slash','blood'],voidwalker:['curse','nova','void'],runesmith:['mark','burst','storm'],alchemist:['poison','burst','toxic']};
      var signatureStyle=signatureStyles[signatureMatch[1]];
      if(signatureStyle && signatureMatch[2]==='finish' && signatureMatch[1]==='elementalist') signatureStyle=['flare','nova','frost'];
      if(signatureStyle && signatureMatch[2]==='finish' && signatureMatch[1]==='alchemist') signatureStyle=['poison','burst','fire'];
      if(signatureStyle){builtin=[name,signatureStyle[signatureMatch[2]==='build'?0:1],signatureStyle[2]];exact=null;}
    }

    var kind = String(option.kind || key.split(':')[0]).toLowerCase();
    var element = visualElement(name, (signatureStyle && signatureStyle[2]) || option.element || (exact && exact.element) || (builtin && builtin[2]));
    var family = builtin && !exact ? builtin[1] : familyFor(name, kind, element);
    var basic = /^(basic_attack|attack|support_strike)$/.test(String(option.ability_id || option.id || ''));
    var weapon = String(option.weapon_type || '').toLowerCase();
    if (basic) {
      if (/^(ranged|crossbow|bow)$/.test(weapon)) family = 'arrow';
      else if (/^(staff|wand)$/.test(weapon)) family = 'bolt';
      else if (weapon === 'spear') family = 'thrust';
      else if (weapon === 'hammer') family = 'quake';
      else if (/^(sword|axe|dagger)$/.test(weapon)) family = 'slash';
    }
    var visualIdentity = key + (basic && weapon ? ':' + weapon + ':' + String(option.weapon_name || '') : '');
    var seed = hash(visualIdentity), motif = verbs.indexOf(String(name).toLowerCase().split(' ')[0]);
    var projectile = /^(arrow|bolt|beam|ray|barrage|volley|breath|wave|drain)$/.test(family);
    var profile = Object.freeze({ key: key, exact: !!exact || !!builtin, name: String(name), kind: kind,
      family: family, element: element, palette: palettes[element], variant: seed,
      signature: seed.toString(16).padStart(8, '0') + hash('rune:' + visualIdentity).toString(16).padStart(8, '0'),
      motif: motif < 0 ? seed % 50 : motif, projectile: projectile,
      sequence: ['prepare', projectile ? 'travel' : 'release', 'impact', 'aftermath'],
      duration: kind === 'ultimate' ? 1040 : family === 'heal' ? 740 : projectile ? 680 : 540,
      pose: /^(strike|slash|thrust|arrow|fang|quake|onslaught)$/.test(family) ? 'attack' : 'cast' });
    return boundedCache(profiles, cacheKey, profile, 4096);
  }
  function actorProfile(unit) {
    unit = unit || {};
    var identity = String(unit.art_key || ((unit.is_player ? 'ally:' : 'monster:') + (unit.name || unit.id || 'unknown')));
    var cacheKey = [identity, unit.name, unit.element, unit.role, unit.is_player, unit.archetype, unit.weapon_type, unit.weapon_name, unit.class, unit.subclass].join('|');
    if (actorProfiles.has(cacheKey)) return actorProfiles.get(cacheKey);
    var subclassIDs=['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'];
    var subIndex=subclassIDs.indexOf(String(unit.subclass||''));
    if(unit.is_player && subIndex>=0){
      var subElements=['holy','blood','frost','nature','fire','arcane','holy','nature','blood','void','storm','toxic'];
      var subElement=subElements[subIndex],subPalette=palettes[subElement],subIdentity='subclass:'+unit.subclass;
      return boundedCache(actorProfiles,cacheKey,{identity:subIdentity,exact:true,rig:unit.subclass,row:subIndex%6,
        asset:'/static/abyss_subclasses_'+(subIndex<6?'martial':'mystic')+'_v1.png',classAtlas:true,
        element:subElement,palette:subPalette,sigil:identityGraphic(subIdentity,subPalette),
        accent:actorAccent(subIdentity,unit.subclass,subPalette),variant:hash(subIdentity),boss:false,portrait:null},512);
    }
    var classRows = {warrior:0,ranger:1,arcanist:2,warden:3,reaver:4,artificer:5};
    var className = String(unit.class || (identity.indexOf('class:') === 0 ? identity.slice(6) : ''));
    if (unit.is_player && Object.prototype.hasOwnProperty.call(classRows, className)) {
      var classElements={warrior:'physical',ranger:'nature',arcanist:'arcane',warden:'holy',reaver:'blood',artificer:'storm'};
      var classElement=classElements[className], classPalette=palettes[classElement];
      return boundedCache(actorProfiles,cacheKey,{identity:'class:'+className,exact:true,rig:className,
        row:classRows[className],asset:'/static/abyss_player_classes_v1.png',classAtlas:true,
        element:classElement,palette:classPalette,sigil:identityGraphic('class:'+className,classPalette),
        accent:actorAccent('class:'+className,className,classPalette),variant:hash(className),boss:false,portrait:null},512);
    }
    var entry = lookup(identity);
    var name = String((entry && entry.name) || unit.name || '').toLowerCase();
    var role = String(unit.role || '').toLowerCase();
    var element = visualElement(name, unit.element || (entry && entry.element)), rig = 'knight';
    // Affixes change identity/accent, never anatomy: a Giant Rat is still a rat.
    name = name.replace(/^(?:(?:snotty|angry|undead|shadow|fiery|ice-cold|toxic|ghostly|metallic|giant)\s+)+(?=rat|slime|goblin|spider|zombie|wolf|skeleton|bat|orc|troll)/, '');
    if (unit.is_player) {
      var loadout = [role, unit.archetype, unit.weapon_type, unit.class].join(' ').toLowerCase();
      if (/ranged|crossbow|bow/.test(String(unit.weapon_type || '').toLowerCase())) rig = 'ranger';
      else if (/staff|wand/.test(String(unit.weapon_type || '').toLowerCase())) rig = /support|heal|druid/.test(loadout) ? 'druid' : 'wizard';
      else if (/dagger/.test(String(unit.weapon_type || '').toLowerCase())) rig = 'rogue';
      else if (/tank|knight|guardian/.test(loadout)) rig = 'knight';
      else if (/ranged|ranger|archer|hunter|bow/.test(loadout)) rig = 'ranger';
      else if (/support|heal|druid|cleric/.test(loadout)) rig = 'druid';
      else if (/rogue|assassin|dagger/.test(loadout)) rig = 'rogue';
      else if (/mage|wizard|caster|staff|wand/.test(loadout) || /arcane|fire|frost|storm|void/.test(element)) rig = 'wizard';
    }
    else if (/gorgoroth/.test(name)) rig = 'gorgoroth';
    else if (/malakor/.test(name)) rig = 'malakor';
    else if (/azazoth/.test(name)) rig = 'azazoth';
    else if (/abyssus/.test(name)) rig = 'abyssus';
    else if (/scribe without eyes/.test(name)) rig = 'scribe';
    else if (/mnemos/.test(name)) rig = 'mnemos';
    else if (/abyss that remembers/.test(name)) rig = 'remembers';
    else if (/gatekeeper/.test(name)) rig = 'gatekeeper';
    else if (/chronos|timekeeper/.test(name)) rig = 'chronos';
    else if (/void.lord/.test(name)) rig = 'void-lord';
    else if (/kraken/.test(name)) rig = 'kraken';
    else if (/\brat\b/.test(name)) rig = 'rat';
    else if (/\bbat\b/.test(name)) rig = 'bat';
    else if (/zombie/.test(name)) rig = 'zombie';
    else if (/skeleton|skeletal/.test(name)) rig = 'skeleton';
    else if (/troll/.test(name)) rig = 'troll';
    else if (/dragon|drake|wyvern|wyrm/.test(name)) rig = 'dragon';
    else if (/spider|arach|scorpion|insect|beetle/.test(name)) rig = 'spider';
    else if (/slime|ooze|blob|jelly/.test(name)) rig = 'slime';
    else if (/golem|construct|sentinel|automaton|colossus|gargoyle|titan|elemental/.test(name)) rig = 'golem';
    else if (/serpent|snake|cobra|basilisk|hydra|leviathan/.test(name)) rig = 'serpent';
    else if (/demon|devil|fiend|balrog|hell|imp/.test(name)) rig = 'demon';
    else if (/ghost|wraith|specter|spectre|banshee|phantom|shade|soul/.test(name)) rig = 'ghost';
    else if (/lich|skeleton|skeletal|bone|necrom|zombie|undead/.test(name)) rig = 'lich';
    else if (/goblin|kobold|gremlin|treasure/.test(name)) rig = 'goblin';
    else if (/orc|ogre|troll|brute|giant|behemoth|minotaur/.test(name)) rig = 'orc';
    else if (/wolf|hound|rat|beast|bear|boar|lion|tiger|cat|bat|raven|eagle|fox|hare/.test(name)) rig = 'wolf';
    else if (/ranger|archer|hunter|scout|bow/.test(name + ' ' + role)) rig = 'ranger';
    else if (/druid|healer|priest|cleric|shaman|support/.test(name + ' ' + role)) rig = 'druid';
    else if (/rogue|assassin|bandit|thief|ninja/.test(name + ' ' + role)) rig = 'rogue';
    else if (/mage|wizard|sorcer|warlock|witch|cultist|caster/.test(name + ' ' + role)) rig = 'wizard';
    else if (entry && (entry.family === 'pets' || entry.family === 'mounts')) rig = 'wolf';
    var index = rigs.indexOf(rig), seed = hash(identity);
    var portrait = (global.AB_EXACT_ICON_MANIFEST || {})[identity] || null;
    var catalogPortrait = !unit.is_player && !!portrait && !!entry && /^(pets|mounts|companions)$/.test(entry.family);
    return boundedCache(actorProfiles, cacheKey, { identity: identity, exact: !!entry, rig: rig, row: index % 8, asset: atlasAssets[Math.floor(index / 8)],
      atlas: Math.floor(index / 8), element: element, palette: palettes[element], sigil: identityGraphic(identity, palettes[element]),
      accent: actorAccent(identity, String((entry && entry.name) || unit.name || '').toLowerCase(), palettes[element]),
      variant: seed, boss: role === 'boss' || !!(entry && entry.family === 'bosses'),
      portrait: portrait, catalogPortrait: catalogPortrait }, 512);
  }
  function actorFrame(unit, pose, frame) {
    var actor = actorProfile(unit), columns = poses[pose] || poses.idle;
    frame = Number.isFinite(Number(frame)) ? Math.max(0, Math.floor(Number(frame))) : 0;
    if (actor.catalogPortrait) {
      var portrait = actor.portrait;
      var transforms = {idle: 'translateY(' + (frame % 2 ? -1 : 0) + 'px)', attack: 'translateX(' + (frame % 2 ? 5 : 2) + 'px) rotate(-3deg)',
        cast: 'translateY(-2px) scale(.98)', hurt: 'rotate(-6deg) scale(.96)', defeat: 'rotate(55deg) scale(.68)'};
      return {asset: portrait.asset, columns: 14, rows: 12, column: portrait.column, row: portrait.row,
        position: (portrait.column * 100 / 13) + '% ' + (portrait.row * 100 / 11) + '%', size: '1400% 1200%',
        transform: transforms[pose] || transforms.idle, rig: 'portrait:' + portrait.family, identity: actor.identity, palette: actor.palette};
    }
    var column = columns[frame % columns.length];
    if(actor.classAtlas) return {asset:actor.asset,columns:8,rows:6,column:column,row:actor.row,
      position:(column*100/7)+'% '+(actor.row*100/5)+'%',size:'800% 600%',
      rig:actor.rig,identity:actor.identity,sigil:actor.sigil,palette:actor.palette};
    var rows = atlasRows[actor.atlas], y = rows[actor.row], height = rows[actor.row + 1] - y;
    return { asset: actor.asset, columns: 8, rows: 8, column: column, row: actor.row,
      position: (column * 100 / 7) + '% ' + (y * 100 / (1254 - height)) + '%', size: '800% ' + (125400 / height) + '%',
      rig: actor.rig, identity: actor.identity, sigil: actor.sigil, palette: actor.palette };
  }
  function svgURL(body, width, height, stretch) {
    return 'data:image/svg+xml,' + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ' + (width || 128) + ' ' + (height || 128) + '"' + (stretch ? ' preserveAspectRatio="none"' : '') + ' fill="none" stroke-linejoin="miter" shape-rendering="crispEdges">' + body + '</svg>');
  }
  function rect(x, y, width, height, color, opacity) {
    return '<rect x="' + x + '" y="' + y + '" width="' + width + '" height="' + height + '" fill="' + color + '" opacity="' + (opacity === undefined ? 1 : opacity) + '"/>';
  }
  function path(d, color, width, fill) {
    return '<path d="' + d + '" stroke="' + color + '" stroke-width="' + (width || 3) + '" fill="' + (fill || 'none') + '"/>';
  }
  function ring(x, y, radius, color, width) {
    return '<circle cx="' + x + '" cy="' + y + '" r="' + radius + '" stroke="' + color + '" stroke-width="' + (width || 3) + '"/>';
  }
  function rune(seed, seed2, colors, x, y, size) {
    var body = '';
    for (var i = 0; i < 64; i++) {
      if (((i < 32 ? seed : seed2) >>> (i % 32)) & 1) body += rect(x + (i % 8) * size, y + Math.floor(i / 8) * size, size, size, colors[i % 3]);
    }
    return body;
  }
  function identityGraphic(identity, colors) {
    return svgURL(rune(hash(identity), hash('rune:' + identity), colors, 0, 0, 2), 16, 16);
  }
  function actorAccent(identity, name, colors) {
    var body = '', c = colors;
    if (/fiery|fire|pyre/.test(name)) {
      for (var f = 0; f < 5; f++) body += path('M' + (21 + f * 20) + ' 110 l-6 -10 l4 -15 l5 8 l5 -5 l-1 15 Z', c[0], 1, c[1] + '77');
    } else if (/ice-cold|frost|icy/.test(name)) {
      for (var ice = 0; ice < 5; ice++) body += path('M' + (20 + ice * 22) + ' 106 l4 -23 l5 8 l-2 15 Z', c[0], 2, c[1] + '55');
    } else if (/toxic|snotty|poison/.test(name)) {
      for (var poison = 0; poison < 6; poison++) body += ring(20 + poison * 18, 99 - (poison % 2) * 9, 3 + poison % 3, c[0], 2);
    } else if (/ghostly|shadow|void/.test(name)) {
      body += path('M18 102 Q3 74 21 49 M109 101 Q125 69 105 42 M30 113 Q62 91 98 111', c[0], 2);
    } else if (/metallic/.test(name)) {
      body += path('M21 92 L12 75 L17 52 M107 92 L116 75 L111 52 M25 109 H103', c[2], 4);
    } else if (/angry|raging/.test(name)) {
      body += path('M25 31 H34 V22 M38 23 V31 H47 M25 36 H34 V44 M38 44 V36 H47', '#ed8e85', 3);
    } else if (/undead/.test(name)) {
      body += path('M19 86 l-6 -16 m0 16 l6 -16 M110 86 l6 -16 m0 16 l-6 -16', c[2], 3);
    } else if (/giant/.test(name)) {
      body += path('M12 109 l10 -9 l10 9 M96 109 l10 -9 l10 9 M36 114 H91', c[0], 3);
    }
    body += rune(hash(identity), hash('rune:' + identity), c, 56, 110, 2);
    return svgURL(body);
  }
  function effectFrame(profile, phase, frame) {
    if (!profile || !profile.family) profile = profileFor(profile);
    frame = Number.isFinite(Number(frame)) ? Math.abs(Math.floor(Number(frame))) % 4 : 0;
    phase = /^(prepare|travel|release|impact|aftermath)$/.test(phase) ? phase : 'impact';
    var cacheKey = [profile.key, profile.signature, profile.family, profile.element, phase, frame].join('|');
    if (effects.has(cacheKey)) return effects.get(cacheKey);
    var c = profile.palette, n = 4 + frame * 3, body = '', family = profile.family;
    var radius = 25 + frame * 7, seed = profile.variant;
    if (phase === 'prepare') {
      body += ring(64, 78, 15 + frame * 4, c[1], 2);
      body += path('M48 76 L56 56 L64 64 L72 44 L80 76', c[0], 3);
    } else if (family === 'restore') {
      body += path('M24 25 L64 12 L104 25 V65 Q95 96 64 116 Q33 96 24 65 Z', c[0], 4, c[1] + '44');
      body += path('M64 38 V82 M42 60 H86', c[2], 7);
      body += ring(64, 65, radius, c[0], 2);
    } else if (family === 'repair') {
      body += path('M30 22 L44 36 L39 49 L26 54 L12 40 Q7 66 32 72 L81 115 L97 99 L53 51 Q62 26 42 16 Z', c[0], 3, c[1] + '77');
      body += path('M77 25 V49 M65 37 H89 M94 65 V81 M86 73 H102', c[2], 3);
    } else if (family === 'potion') {
      body += path('M51 16 H77 V43 Q100 57 100 81 Q100 110 64 110 Q28 110 28 81 Q28 57 51 43 Z', c[0], 4, c[1] + '55');
      body += path('M49 23 H79 M34 78 Q49 68 64 78 T94 78', c[2], 3);
      for (var drop = 0; drop < 3; drop++) body += ring(49 + drop * 15, 91 - frame * 5 - drop * 6, 3, c[2], 2);
    } else if (family === 'heal') {
      body += ring(64, 96, 27 + frame * 3, c[1], 2);
      for (var h = 0; h < 3; h++) {
        var hx = 30 + h * 29, hy = 79 - frame * 9 - (h % 2) * 16;
        body += path('M' + hx + ' ' + (hy - 8) + ' v16 M' + (hx - 8) + ' ' + hy + ' h16', c[h], 5);
      }
      body += path('M39 91 Q16 60 41 34 M89 91 Q111 60 88 34', c[0], 3);
    } else if (family === 'revive') {
      body += path('M64 109 V40 M63 78 Q18 87 13 37 L37 49 L30 23 L61 56 M65 78 Q110 87 115 37 L91 49 L98 23 L67 56', c[0], 5);
      body += path('M49 38 L64 16 L79 38 L64 32 Z', c[2], 3, c[1] + '77');
    } else if (family === 'poison') {
      for (var bubble = 0; bubble < 7; bubble++) body += ring(27 + bubble * 12, 97 - (bubble % 3) * 17 - frame * 3, 4 + bubble % 4, c[bubble % 3], 3);
      body += path('M44 69 L64 25 L85 69 Q93 99 64 99 Q35 99 44 69 Z', c[0], 3, c[1] + '55');
    } else if (family === 'reflect') {
      body += path('M20 22 L60 63 L20 103 M61 20 V108 M113 31 L79 63 L113 97', c[0], 4);
      body += path('M82 65 H111 l-10 -9 m10 9 l-10 9', c[2], 3);
    } else if (family === 'silence' || family === 'fatigue') {
      body += ring(64, 64, 34, c[1], 4);
      body += path(family === 'silence' ? 'M38 37 L90 91 M47 69 V54 L61 43 L79 44 L80 76' : 'M43 43 H80 L44 86 H82 M43 58 H77', c[0], 5);
    } else if (family === 'warning' || family === 'enrage') {
      body += path('M64 17 L112 105 H16 Z', c[0], 3, c[1] + '33');
      if (family === 'warning') body += path('M64 43 V76 M64 88 V93', c[2], 6);
      else body += path('M38 59 L53 69 L58 61 M90 59 L75 69 L70 61 M43 92 L52 80 L64 88 L76 80 L85 92', c[2], 4);
    } else if (family === 'shield') {
      body += path('M64 18 L99 32 L94 77 L80 95 L64 106 L48 95 L34 77 L29 32 Z', c[0], 4, c[1] + '33');
      body += path('M64 31 V90 M42 43 H86 M44 65 H84', c[2], 2);
    } else if (family === 'arrow' || family === 'thrust') {
      if (phase === 'travel') {
        body += path('M10 64 H117 M99 51 L117 64 L99 77 M20 53 L30 64 L20 75', c[2], 4);
        body += path('M7 84 H48 M5 94 H31', c[0], 3);
      } else {
        body += path('M15 77 L104 48 M83 45 L106 48 L91 65 M21 64 L31 72 L26 86', c[2], 4);
        body += path('M8 89 L45 78 M14 101 L57 84', c[0], 3);
      }
    } else if (family === 'lightning' || family === 'storm') {
      body += path('M16 18 L49 35 L37 54 L78 58 L65 79 L113 104 M49 35 L86 27 L93 45 M65 79 L35 101', c[1], 9);
      body += path('M16 18 L49 35 L37 54 L78 58 L65 79 L113 104 M49 35 L86 27 L93 45 M65 79 L35 101', c[2], 3);
    } else if (family === 'slash' || family === 'onslaught' || family === 'fury' || family === 'rage') {
      for (var s = 0; s < (family === 'slash' ? 2 : 4); s++) body += path('M' + (19 + s * 8) + ' 105 Q' + (48 + s * 10) + ' 45 ' + (109 - s * 5) + ' ' + (15 + s * 8), c[s % 3], 5 - s % 3);
    } else if (family === 'fang') {
      body += path('M23 34 L44 86 L52 46 M106 34 L85 86 L77 46 M32 91 L50 73 M96 91 L78 73', c[2], 4);
    } else if (family === 'beam' || family === 'ray') {
      body += path('M8 64 H120', c[1], family === 'beam' ? 22 : 10);
      body += path('M8 64 H120', c[2], 4);
      body += path('M18 44 L38 64 L18 84 M83 40 L103 64 L83 88', c[0], 3);
    } else if (family === 'barrage' || family === 'volley') {
      for (var v = 0; v < 5; v++) {
        var y = 20 + v * 21, start = 12 + (v % 2) * 15;
        body += path('M' + start + ' ' + y + ' L' + (start + 62) + ' ' + (y + 10) + ' l-13 -9 m13 9 l-15 5', c[v % 3], family === 'volley' ? 3 : 5);
      }
    } else if (family === 'curse' || family === 'mark') {
      body += ring(64, 60, radius, c[0], 2);
      body += path('M25 60 H43 M85 60 H103 M64 21 V39 M64 81 V99', c[2], 3);
      if (family === 'curse') body += path('M45 43 L54 72 L64 63 L74 72 L83 43 M52 79 H76', c[1], 5);
      else body += path('M49 61 L60 72 L82 47', c[2], 4);
    } else if (family === 'drain') {
      body += path('M109 30 Q14 21 62 62 Q110 103 18 98 M106 42 Q37 31 68 66 Q96 94 25 88', c[0], 4);
      for (var d = 0; d < 5; d++) body += rect(22 + d * 19, 40 + ((d + frame) % 3) * 15, 5, 5, c[2]);
    } else if (family === 'quake') {
      body += path('M10 99 L33 80 L44 86 L63 51 L68 81 L89 66 L117 100 M63 51 L45 31 M68 81 L82 105', c[0], 5);
      for (var q = 0; q < 7; q++) body += rect(16 + q * 15, 66 - (q % 3) * n, 8, 6, c[q % 3]);
    } else if (family === 'wave' || family === 'surge' || family === 'breath') {
      for (var w = 0; w < 3; w++) body += path('M' + (24 + w * 16) + ' 104 Q' + (87 + frame * 3) + ' 64 ' + (24 + w * 16) + ' 24', c[w], family === 'breath' ? 11 : 5);
    } else if (family === 'nova' || family === 'pulse' || family === 'summon') {
      body += ring(64, 64, radius, c[0], 4);
      body += ring(64, 64, Math.max(8, radius - 13), c[1], 2);
      if (family === 'summon') body += path('M64 23 L89 95 L26 49 L102 49 L39 95 Z', c[2], 2);
      else for (var p = 0; p < 8; p++) body += '<g transform="rotate(' + p * 45 + ' 64 64)">' + path('M64 8 V23', c[2], 3) + '</g>';
    } else if (family === 'bolt') {
      body += path('M14 91 L53 64 L45 53 L106 29 L81 59 L91 68 Z', c[0], 3, c[1]);
      body += path('M28 89 L66 63 L60 54 L96 36', c[2], 3);
    } else if (family === 'flare') {
      body += path('M64 109 Q17 83 47 47 L57 62 L70 15 L80 50 L98 35 Q118 87 64 109 Z', c[0], 3, c[1] + '77');
      body += path('M61 95 Q43 79 68 58 Q86 87 61 95 Z', c[2], 3);
    } else {
      var points = [], count = family === 'burst' ? 12 : family === 'wrath' ? 7 : family === 'blast' ? 10 : 6;
      for (var b = 0; b < count * 2; b++) {
        var theta = b * Math.PI / count, r = b % 2 ? 15 : radius;
        points.push(Math.round(64 + Math.cos(theta) * r) + ',' + Math.round(64 + Math.sin(theta) * r));
      }
      body += '<polygon points="' + points.join(' ') + '" fill="' + c[1] + '55" stroke="' + c[0] + '" stroke-width="4"/>';
      body += rect(59, 59, 10, 10, c[2]);
    }
    // Prefix motifs change silhouette, direction and rhythm independently of element color.
    var arms = 3 + profile.motif % 5, offset = (profile.motif % 10) * 9 + frame * 7;
    for (var a = 0; a < arms; a++) {
      var angle = (offset + a * 360 / arms) * Math.PI / 180;
      var x = Math.round(64 + Math.cos(angle) * (47 + profile.motif % 6));
      var y2 = Math.round(64 + Math.sin(angle) * (47 + profile.motif % 6));
      if (profile.motif % 3 === 0) body += path('M' + (x - 3) + ' ' + y2 + ' l3 -7 l3 7 l-3 5 Z', c[1], 2);
      else if (profile.motif % 3 === 1) body += path('M' + (x - 5) + ' ' + y2 + ' h10 M' + x + ' ' + (y2 - 5) + ' v10', c[0], 2);
      else body += rect(x - 2, y2 - 2, 4 + profile.motif % 4, 4, c[2]);
    }
    // An exact 64-bit rune makes catalog rarity/name duplicates visually distinguishable.
    body += rune(seed, hash('rune:' + profile.key), c, 56, 105, 2);
    if (phase === 'aftermath') body = '<g opacity="0.48">' + body + '</g>';
    return boundedCache(effects, cacheKey, svgURL(body, 128, 128, phase === 'travel'), 256);
  }

  var biomeStyles = {
    'Mossbound': ['nature', 'moss'], 'Cinder-Choked': ['fire', 'chimneys'], 'Fogbound': ['water', 'fog'],
    'Rootbound': ['nature', 'roots'], 'Bloodrust': ['blood', 'chains'], 'Frostbitten': ['frost', 'crystals'],
    'Storm-Wracked': ['storm', 'spires'], 'Venom-Veiled': ['toxic', 'fungus'], 'Pyre-Eternal': ['fire', 'pyres'],
    'Voidscarred': ['void', 'rifts'], 'Starless': ['shadow', 'eclipse'], 'Soul-Rent': ['holy', 'souls'],
    'Oblivion-Touched': ['void', 'fractures']
  };
  function backdrop(options) {
    options = options || {};
    var key = String(options.biome || options.biome_name || '');
    Object.keys(biomeStyles).some(function (candidate) {
      if (key.toLowerCase().indexOf(candidate.toLowerCase()) >= 0) { key = candidate; return true; }
      return false;
    });
    if (!biomeStyles[key]) key = Number(options.depth) > 150 ? 'Voidscarred' : Number(options.depth) > 50 ? 'Frostbitten' : 'Mossbound';
    if (backdropCache.has(key)) return backdropCache.get(key);
    var style = biomeStyles[key], c = palettes[style[0]], type = style[1];
    var body = rect(0, 0, 640, 240, '#111923') + rect(0, 165, 640, 75, '#19212b');
    for (var i = 0; i < 13; i++) {
      var x = i * 56 - 16, height = 42 + hash(key + i) % 80;
      body += path('M' + x + ' 165 V' + (165 - height) + ' l14 -15 h22 l12 15 V165 Z', '#28303b', 0, '#202933');
      if (type === 'crystals' || type === 'spires') body += path('M' + (x + 10) + ' 163 l8 -' + height + ' l12 18 l8 ' + (height - 18) + ' Z', c[1], 1, c[1] + '33');
      if (type === 'moss' || type === 'roots') body += path('M' + x + ' 30 Q' + (x + 38) + ' 75 ' + (x + 17) + ' 110 Q' + x + ' 150 ' + (x + 40) + ' 172', c[1] + '77', type === 'roots' ? 7 : 3);
      if (type === 'fungus') body += path('M' + (x + 20) + ' 170 V143 m-15 0 Q' + (x + 20) + ' 111 ' + (x + 37) + ' 143 Z', c[0] + '99', 3, c[1] + '33');
      if (type === 'chains') for (var link = 0; link < 6; link++) body += ring(x + 23, link * 17 + 15, 7, '#634348', 2);
      if (type === 'chimneys' || type === 'pyres') {
        body += rect(x + 10, 120, 22, 47, '#493439');
        body += path('M' + (x + 8) + ' 122 l13 -35 l4 17 l12 -9 l-6 29 Z', c[0] + '88', 1, c[1] + '55');
      }
      if (type === 'rifts' || type === 'fractures') body += path('M' + x + ' 26 l35 44 l-15 32 l34 49', c[1] + '88', type === 'rifts' ? 3 : 1);
      if (type === 'souls') body += path('M' + (x + 16) + ' 98 Q' + (x - 2) + ' 60 ' + (x + 18) + ' 57 Q' + (x + 40) + ' 60 ' + (x + 28) + ' 89 l-6 -7 Z', c[0] + '55', 1, c[1] + '22');
      body += rect(x + 7, 179 + hash(i + key) % 44, 25, 2, c[1] + '33');
    }
    if (type === 'fog') for (var fog = 0; fog < 4; fog++) body += path('M0 ' + (102 + fog * 20) + ' Q160 ' + (69 + fog * 20) + ' 320 ' + (102 + fog * 20) + ' T640 ' + (102 + fog * 20), c[0] + '22', 13);
    if (type === 'eclipse') body += '<circle cx="320" cy="75" r="45" fill="#131220" stroke="#75638d" stroke-width="2"/>';
    var result = Object.freeze({ key: key, image: svgURL(body, 640, 240), palette: c, element: style[0] });
    backdropCache.set(key, result);
    return result;
  }

  global.AbyssCombatArt = Object.freeze({ version: 2, profileFor: profileFor, actorProfile: actorProfile,
    actorFrame: actorFrame, effectFrame: effectFrame, backdrop: backdrop,
    rigs: Object.freeze(rigs.slice()), poses: Object.freeze(poses), atlasAssets: Object.freeze(atlasAssets.slice()),
    builtinIDs: Object.freeze(Object.keys(builtins)) });
})(window);
