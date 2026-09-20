const { test, expect } = require('@playwright/test');

test.describe('Use distance attenuation for enemy sounds (Proposal 0188)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('RiftAudio.distanceAttenuation computes expected rolloff curve', async ({ page }) => {
    const results = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        close: audio.distanceAttenuation(50),
        threshold: audio.distanceAttenuation(120),
        mid: audio.distanceAttenuation(485),
        far: audio.distanceAttenuation(850),
        beyond: audio.distanceAttenuation(1200),
        zero: audio.distanceAttenuation(0),
        negative: audio.distanceAttenuation(-100),
        invalid: audio.distanceAttenuation(null),
      };
    });

    expect(results.close).toBe(1.0);
    expect(results.threshold).toBe(1.0);
    expect(results.mid).toBeCloseTo(0.65, 2);
    expect(results.far).toBe(0.3);
    expect(results.beyond).toBe(0.3);
    expect(results.zero).toBe(1.0);
    expect(results.negative).toBe(1.0);
    expect(results.invalid).toBe(1.0);
  });

  test('RiftAudio.isEnemyCue correctly identifies enemy cues vs player/interface cues', async ({ page }) => {
    const checks = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        goblinAttack: audio.isEnemyCue('goblin_attack'),
        knightAttack: audio.isEnemyCue('knight_attack'),
        bossRoar: audio.isEnemyCue('boss_roar'),
        slam: audio.isEnemyCue('slam'),
        arrow: audio.isEnemyCue('arrow'),
        treasureEscape: audio.isEnemyCue('treasure_escape'),
        goblinHurt: audio.isEnemyCue('goblin_hurt'),
        bossHurt: audio.isEnemyCue('boss_hurt'),
        knightDeath: audio.isEnemyCue('knight_death'),
        wolfDeath: audio.isEnemyCue('wolf_death'),
        sporeDeath: audio.isEnemyCue('spore_death'),
        // Player / UI / General sounds
        step: audio.isEnemyCue('step'),
        jump: audio.isEnemyCue('jump'),
        land: audio.isEnemyCue('land'),
        slash: audio.isEnemyCue('slash'),
        block: audio.isEnemyCue('block'),
        pickup: audio.isEnemyCue('pickup'),
        ui: audio.isEnemyCue('ui'),
        bank: audio.isEnemyCue('bank'),
      };
    });

    // Enemy cues
    expect(checks.goblinAttack).toBe(true);
    expect(checks.knightAttack).toBe(true);
    expect(checks.bossRoar).toBe(true);
    expect(checks.slam).toBe(true);
    expect(checks.arrow).toBe(true);
    expect(checks.treasureEscape).toBe(true);
    expect(checks.goblinHurt).toBe(true);
    expect(checks.bossHurt).toBe(true);
    expect(checks.knightDeath).toBe(true);
    expect(checks.wolfDeath).toBe(true);
    expect(checks.sporeDeath).toBe(true);

    // Player / system cues
    expect(checks.step).toBe(false);
    expect(checks.jump).toBe(false);
    expect(checks.land).toBe(false);
    expect(checks.slash).toBe(false);
    expect(checks.block).toBe(false);
    expect(checks.pickup).toBe(false);
    expect(checks.ui).toBe(false);
    expect(checks.bank).toBe(false);
  });

  test('RiftAudio.play synthesizes enemy cues with distance attenuation', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      // Close enemy sound (full volume)
      audio.play('goblin_attack', 0, null, null, 80);

      // Mid-distance enemy sound
      audio.play('boss_roar', 0.4, null, null, 400);

      // Far enemy sound
      audio.play('knight_death', -0.6, null, null, 900);

      // Close hurt cue
      audio.hurt('goblin', 0, 50);

      // Far hurt cue
      audio.hurt('knight', 0.5, 750);

      // Close death cue
      audio.death('boss', 0, 100);

      // Far death cue
      audio.death('wolf', -0.5, 800);

      const after = audio.played;
      return { before, after };
    });

    expect(result.after).toBe(result.before + 7);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('renderer snapshot calculates distance and forwards it to RiftAudio.play', async ({ page }) => {
    const playback = await page.evaluate(async () => {
      const calls = [];
      const origPlay = window.RiftAudio.play;
      window.RiftAudio.play = function (kind, pan, extra, extra2, distance) {
        calls.push({ kind, pan, extra, extra2, distance });
        return origPlay.apply(this, arguments);
      };

      const mockRun = {
        id: 'test-distance-run',
        counter: 40,
        room: 0,
        status: 'fighting',
        build: { weapon: 'Void Katana', class: 'bloodblade' },
        player: { id: 'player', x: 200, y: 350, hp: 100, max_hp: 100, mana: 50, facing: 1, pose: 'idle', jump: 0 },
        enemies: [
          { id: 'near-goblin', kind: 'goblin', x: 280, y: 350, hp: 50, max_hp: 100 },
          { id: 'far-boss', kind: 'boss', x: 800, y: 350, hp: 500, max_hp: 500 },
        ],
        projectiles: [],
        drops: [],
        events: [
          // Near event: dx = 80, dy = 0, dist = 80
          { id: 40, kind: 'goblin_attack', x: 280, y: 350, value: 0 },
          // Far event: dx = 600, dy = 0, dist = 600
          { id: 41, kind: 'boss_roar', x: 800, y: 350, value: 0 },
        ]
      };

      window.RiftRenderer.snapshot(mockRun, false);
      window.RiftAudio.play = origPlay;
      return calls;
    });

    const nearCall = playback.find(c => c.kind === 'goblin_attack');
    expect(nearCall).toBeDefined();
    expect(nearCall.distance).toBeCloseTo(80, 1);
    expect(nearCall.pan).toBeCloseTo(80 / 700, 2);

    const farCall = playback.find(c => c.kind === 'boss_roar');
    expect(farCall).toBeDefined();
    expect(farCall.distance).toBeCloseTo(600, 1);
    expect(farCall.pan).toBeCloseTo(600 / 700, 2);
  });
});
