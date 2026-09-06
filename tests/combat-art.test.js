const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const test = require('node:test');

const assets = path.join(__dirname, '..', 'internal', 'bot', 'webassets');
const context = { window: {} };
vm.createContext(context);
vm.runInContext(fs.readFileSync(path.join(assets, 'abyss_catalog_icons.js'), 'utf8'), context);
vm.runInContext(fs.readFileSync(path.join(assets, 'abyss_combat_catalog.js'), 'utf8'), context);
vm.runInContext(fs.readFileSync(path.join(assets, 'abyss_combat_art.js'), 'utf8'), context);
const art = context.window.AbyssCombatArt;
const catalog = context.window.AB_COMBAT_CATALOG;

test('named elemental skills retain their visual theme with broad engine elements', () => {
  for (const [name, element] of [['Fiery Bolt', 'fire'], ['Icy Bolt', 'frost'], ['Shadow Strike', 'shadow'], ['Storm Blast', 'storm'], ['Blood Drain', 'blood']]) {
    const key = Object.keys(catalog).find(key => catalog[key].name === name);
    assert.ok(key, name);
    assert.equal(art.profileFor({art_key: key, element: catalog[key].element}).element, element, name);
  }
});

test('element names do not replace an ability action and item effects have suitable forms', () => {
  for (const [name, family] of [['Storm Heal', 'heal'], ['Storm Shield', 'shield'], ['Blood Drain', 'drain'], ['Fiery Sunder', 'quake'], ['Arcane Mend', 'heal']]) {
    const key = Object.keys(catalog).find(key => catalog[key].name === name);
    assert.equal(art.profileFor({art_key: key}).family, family, name);
  }
  assert.equal(art.profileFor({art_key: 'item:repair_kit'}).family, 'repair');
  assert.equal(art.profileFor({art_key: 'item:phoenix_feather'}).family, 'revive');
  assert.equal(art.profileFor({art_key: 'item:abyss_emergency_revive'}).family, 'revive');
});

test('every existing ability has an exact stable, distinct graphic and bounded animation', () => {
  const keys = Object.keys(catalog).filter(key => /^(skill:|ultimate:)/.test(key));
  assert.ok(keys.length > 1000);
  const graphics = new Set();
  for (const key of keys) {
    const profile = art.profileFor({ art_key: key });
    assert.equal(profile.key, key);
    assert.equal(profile.exact, true);
    assert.ok(profile.duration > 0 && profile.duration <= 1200);
    const svg = art.effectFrame(profile, 'impact', 1);
    assert.ok(svg.startsWith('data:image/svg+xml,'), key);
    assert.ok(svg.length < 24000, key);
    assert.equal(svg, art.effectFrame(profile, 'impact', 1));
    assert.ok(!svg.includes('NaN'), key);
    assert.ok(!graphics.has(svg), `${key} reuses another ability graphic`);
    graphics.add(svg);
  }
});

test('actors preserve exact identities while using actual pose columns', () => {
  const keys = Object.keys(catalog).filter(key => /^(monster:|pet-type:)/.test(key) || /^(pets|companions|mounts)$/.test(catalog[key].family));
  assert.ok(keys.length > 50);
  const accents = new Set();
  for (const key of keys) {
    const unit = { art_key: key, name: catalog[key].name };
    assert.equal(art.actorProfile(unit).identity, key);
    const accent = art.actorProfile(unit).accent;
    assert.ok(!accents.has(accent), `${key} duplicates an actor identity`);
    accents.add(accent);
    const poses = ['idle', 'attack', 'cast', 'hurt', 'defeat'].map(pose => {
      const frame = art.actorFrame(unit, pose, 1);
      return frame.position + '|' + (frame.transform || '');
    });
    assert.equal(new Set(poses).size, 5, key);
    for (const pose of ['idle', 'attack', 'cast', 'hurt', 'defeat']) {
      const frame = art.actorFrame(unit, pose, 0);
      assert.ok(frame.column >= 0 && frame.column < frame.columns);
      assert.ok(frame.row >= 0 && frame.row < frame.rows);
    }
  }
});

test('generic companion equipment animates its own portrait instead of turning into a wolf', () => {
  const manifest = context.window.AB_EXACT_ICON_MANIFEST;
  for (const key of Object.keys(catalog).filter(key => /^(pets|mounts|companions)$/.test(catalog[key].family))) {
    for (const pose of ['idle', 'attack', 'cast', 'hurt', 'defeat']) {
      const frame = art.actorFrame({art_key: key}, pose, 0);
      assert.equal(frame.asset, manifest[key].asset, key);
      assert.equal(frame.column, manifest[key].column, key);
      assert.equal(frame.row, manifest[key].row, key);
    }
  }
});

test('legacy artifact keys and class finisher themes resolve to their current exact identity', () => {
  assert.equal(art.profileFor({art_key: 'item:artifact:0:Corrupted Soul'}).key, 'item:artifact:0');
  assert.equal(art.profileFor({art_key: 'item:artifact:0:Corrupted Soul'}).exact, true);
  assert.equal(art.profileFor({art_key: 'skill:CLASS_elementalist_finish'}).element, 'frost');
  assert.equal(art.profileFor({art_key: 'skill:CLASS_alchemist_finish'}).element, 'fire');
});

test('companion commands and active relics show the resolved support action', () => {
  for (const [id, family] of [['focus', 'mark'], ['guard', 'shield'], ['free', 'fang']]) {
    const profile = art.profileFor({kind: 'companion', ability_id: id, ability_name: 'Companion command'});
    assert.equal(profile.exact, true);
    assert.equal(profile.family, family);
  }
  assert.equal(art.profileFor({kind:'relic', id:'ABYSS_RELIC', name:'Heart of the Abyss'}).family, 'restore');
});

test('all item, relic, companion, artifact and consumable identities have distinct effects', () => {
  const graphics = new Set();
  const keys = Object.keys(catalog).filter(key => key.startsWith('item:'));
  assert.ok(keys.length > 1400);
  for (const key of keys) {
    const profile = art.profileFor({ art_key: key, kind: 'item' });
    assert.equal(profile.exact, true);
    const graphic = art.effectFrame(profile, 'impact', 0);
    assert.ok(!graphics.has(graphic), `${key} duplicates an item effect`);
    graphics.add(graphic);
  }
});

test('effect forms communicate healing, arrows, shielding and lightning differently', () => {
  assert.equal(art.profileFor({ kind: 'skill', name: 'Divine Heal' }).family, 'heal');
  assert.equal(art.profileFor({ kind: 'skill', name: 'Piercing Arrow' }).family, 'arrow');
  assert.equal(art.profileFor({ kind: 'skill', name: 'Arcane Shield' }).family, 'shield');
  assert.equal(art.profileFor({ kind: 'skill', name: 'Chain Lightning' }).family, 'lightning');
  assert.equal(art.profileFor({ kind: 'skill', name: 'Divine Heal' }).projectile, false);
  assert.equal(art.profileFor({ kind: 'skill', name: 'Piercing Arrow' }).projectile, true);
  assert.equal(art.profileFor({ name: 'Unknown', element: 'Air' }).element, 'air');
  assert.equal(art.profileFor({ name: 'Unknown', element: 'Earth' }).element, 'nature');
});

test('monster affixes and player names do not change actor anatomy', () => {
  for (const name of ['Undead Rat', 'Giant Rat', 'Ghostly Rat']) assert.equal(art.actorProfile({ name }).rig, 'rat');
  assert.equal(art.actorProfile({ name: 'Ghostly Orc' }).rig, 'orc');
  assert.equal(art.actorProfile({ name: 'Dragon', is_player: true, role: 'tank' }).rig, 'knight');
  assert.equal(art.actorProfile({ name: 'Clockmaster', art_key: 'monster:109' }).rig, 'chronos');
  assert.equal(art.actorProfile({ is_player: true, weapon_type: 'ranged' }).rig, 'ranger');
  assert.equal(art.actorProfile({ is_player: true, weapon_type: 'staff' }).rig, 'wizard');
  assert.equal(art.actorProfile({ is_player: true, weapon_type: 'wand' }).rig, 'wizard');
});

test('authoritative equipped weapons produce distinct basic attacks without cache cross-contamination', () => {
  const graphics = new Set();
  for (const [weapon_type, family] of [['ranged', 'arrow'], ['staff', 'bolt'], ['sword', 'slash'], ['hammer', 'quake'], ['spear', 'thrust']]) {
    const profile = art.profileFor({ ability_id: 'basic_attack', kind: 'attack', weapon_type });
    assert.equal(profile.family, family);
    const graphic = art.effectFrame(profile, 'travel', 0);
    assert.ok(!graphics.has(graphic));
    graphics.add(graphic);
  }
  const arrow = art.profileFor({ ability_id: 'basic_attack', kind: 'attack', weapon_type: 'ranged' });
  assert.equal(arrow.family, 'arrow');
  assert.equal(arrow.projectile, true);
  assert.ok(decodeURIComponent(art.effectFrame(arrow, 'travel', 0)).includes('preserveAspectRatio="none"'));
});

test('every server pet, passive, status and boss phase has its own explicit effect', () => {
  const effects = new Set();
  assert.ok(art.builtinIDs.length >= 35);
  for (const id of art.builtinIDs) {
    const profile = art.profileFor({ id, kind: 'trigger' });
    assert.equal(profile.exact, true, id);
    const graphic = art.effectFrame(profile, 'impact', 0);
    assert.ok(!effects.has(graphic), `${id} duplicates an event effect`);
    effects.add(graphic);
  }
  assert.equal(art.profileFor({ id: 'pet_mending_cry', kind: 'companion' }).family, 'heal');
  assert.equal(art.profileFor({ id: 'death_explosion', kind: 'trigger' }).family, 'nova');
  assert.equal(art.profileFor({ id: 'enrage_warning', kind: 'phase' }).family, 'warning');
});

test('all seven named Abyss bosses have individual authored animation rigs', () => {
  const bosses = Object.keys(catalog).filter(key => catalog[key].variant === 'AbyssBoss');
  assert.equal(bosses.length, 7);
  const rigs = new Set();
  for (const art_key of bosses) {
    const profile = art.actorProfile({ art_key });
    assert.equal(profile.exact, true);
    assert.equal(profile.boss, true);
    assert.ok(profile.asset.endsWith('abyss_combat_bosses_v2.png'));
    rigs.add(profile.rig);
  }
  assert.equal(rigs.size, 7);
});

test('all thirteen biomes have distinct scenery and unknown inputs have safe fallbacks', () => {
  const names = ['Mossbound', 'Cinder-Choked', 'Fogbound', 'Rootbound', 'Bloodrust', 'Frostbitten', 'Storm-Wracked', 'Venom-Veiled', 'Pyre-Eternal', 'Voidscarred', 'Starless', 'Soul-Rent', 'Oblivion-Touched'];
  assert.equal(new Set(names.map(biome => art.backdrop({ biome }).image)).size, 13);
  const unknown = art.profileFor({ id: '<script>arbitrary</script>', name: 'Unknown Future Ability' });
  assert.equal(unknown.exact, false);
  assert.ok(!decodeURIComponent(art.effectFrame(unknown, 'impact', 0)).includes('<script>'));
  assert.ok(art.actorFrame({}, 'nonexistent', NaN).asset);
});

test('raster atlas assets exist and stay inside the download budget', () => {
  let bytes = 0;
  for (const filename of ['abyss_combat_roles_v2.png', 'abyss_combat_creatures_v2.png', 'abyss_combat_bestiary_v2.png', 'abyss_combat_bosses_v2.png']) {
    const image = fs.readFileSync(path.join(assets, filename));
    assert.equal(image.subarray(1, 4).toString(), 'PNG');
    // CSS uses proportional atlas coordinates, including fractional source pixels.
    assert.ok(image.readUInt32BE(16) >= 1024);
    assert.equal(image.readUInt32BE(16), image.readUInt32BE(20));
    bytes += image.length;
  }
  assert.ok(bytes < 14000000, `Atlas total ${bytes} exceeds 14 MB budget`);
});


test('all classes and empowered subclasses own distinct rigs across every pose', () => {
 const classes=['warrior','ranger','arcanist','warden','reaver','artificer'];
 const subs=['vanguard','berserker','marksman','beastmaster','elementalist','chronomancer','oracle','geomancer','bloodblade','voidwalker','runesmith','alchemist'];
 const frames=new Set();
 for(const id of classes.concat(subs)) {
  const unit={is_player:true,class:classes.includes(id)?id:classes[Math.floor(subs.indexOf(id)/2)],subclass:subs.includes(id)?id:'',weapon_type:'staff'};
  for(const [pose,count]of Object.entries({idle:2,attack:2,cast:2,hurt:1,defeat:1}))for(let i=0;i<count;i++) {
   const frame=art.actorFrame(unit,pose,i);assert.equal(frame.rig,id);assert.ok(fs.existsSync(path.join(assets,path.basename(frame.asset))));frames.add(frame.asset+'|'+frame.position);
  }
 }
 assert.equal(frames.size,144);
 for(const sub of subs)for(const role of ['build','finish']){
  const profile=art.profileFor({kind:'skill',ability_id:'CLASS_'+sub+'_'+role,element:'physical'});assert.equal(profile.exact,true);assert.ok(art.effectFrame(profile,'impact',0).startsWith('data:image/svg+xml,'));
 }
});
