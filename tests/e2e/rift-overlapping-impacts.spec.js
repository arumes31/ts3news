const { test, expect } = require('@playwright/test');

test.describe('Reduce overlapping identical impact sounds (Proposal 0189)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('RiftAudio.isImpactCue identifies impact cues vs other cues', async ({ page }) => {
    const checks = await page.evaluate(() => {
      const audio = window.RiftAudio;
      return {
        hit: audio.isImpactCue('hit'),
        hitBlade: audio.isImpactCue('hit_blade'),
        hitBlunt: audio.isImpactCue('hit_blunt'),
        hitPierce: audio.isImpactCue('hit_pierce'),
        hitArcane: audio.isImpactCue('hit_arcane'),
        hitFist: audio.isImpactCue('hit_fist'),
        hitRanged: audio.isImpactCue('hit_ranged'),
        slash: audio.isImpactCue('slash'),
        slam: audio.isImpactCue('slam'),
        block: audio.isImpactCue('block'),
        perfectGuard: audio.isImpactCue('perfect_guard'),
        knockdown: audio.isImpactCue('knockdown'),
        goblinHurt: audio.isImpactCue('goblin_hurt'),
        // Non-impact cues
        step: audio.isImpactCue('step'),
        jump: audio.isImpactCue('jump'),
        land: audio.isImpactCue('land'),
        ui: audio.isImpactCue('ui'),
        bank: audio.isImpactCue('bank'),
        defeat: audio.isImpactCue('defeat'),
      };
    });

    expect(checks.hit).toBe(true);
    expect(checks.hitBlade).toBe(true);
    expect(checks.hitBlunt).toBe(true);
    expect(checks.hitPierce).toBe(true);
    expect(checks.hitArcane).toBe(true);
    expect(checks.hitFist).toBe(true);
    expect(checks.hitRanged).toBe(true);
    expect(checks.slash).toBe(true);
    expect(checks.slam).toBe(true);
    expect(checks.block).toBe(true);
    expect(checks.perfectGuard).toBe(true);
    expect(checks.knockdown).toBe(true);
    expect(checks.goblinHurt).toBe(true);

    expect(checks.step).toBe(false);
    expect(checks.jump).toBe(false);
    expect(checks.land).toBe(false);
    expect(checks.ui).toBe(false);
    expect(checks.bank).toBe(false);
    expect(checks.defeat).toBe(false);
  });

  test('audio throttles excessive identical impact sounds within rapid interval', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      audio.clearImpactHistory();
      await audio.setActive(true);

      const before = audio.played;

      // Fire 10 identical hit_blade impacts in the exact same millisecond
      for (let i = 0; i < 10; i++) {
        audio.play('hit_blade', 0);
      }

      const countAfterBlade = audio.played - before;

      // Now fire 5 DISTINCT impact families
      const beforeDistinct = audio.played;
      audio.play('hit_blunt', 0);
      audio.play('hit_pierce', 0);
      audio.play('hit_arcane', 0);
      audio.play('hit_fist', 0);
      audio.play('hit_ranged', 0);
      const countAfterDistinct = audio.played - beforeDistinct;

      return { countAfterBlade, countAfterDistinct };
    });

    // Only up to 2 identical impacts should play, throttling the rest
    expect(results.countAfterBlade).toBe(2);

    // All 5 distinct impact families should play without throttling each other
    expect(results.countAfterDistinct).toBe(5);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('identical impact sounds play again after throttling cooldown elapses', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      audio.clearImpactHistory();
      await audio.setActive(true);

      const before = audio.played;

      // Burst 1: 5 identical hits
      for (let i = 0; i < 5; i++) {
        audio.play('hit_blunt', 0);
      }
      const burst1Count = audio.played - before;

      // Wait 60ms for cooldown to elapse
      await new Promise(r => setTimeout(r, 60));

      const beforeBurst2 = audio.played;
      // Burst 2: 5 identical hits
      for (let i = 0; i < 5; i++) {
        audio.play('hit_blunt', 0);
      }
      const burst2Count = audio.played - beforeBurst2;

      return { burst1Count, burst2Count };
    });

    expect(results.burst1Count).toBe(2);
    expect(results.burst2Count).toBe(2);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('renderer snapshot throttles multiple identical impact events from simultaneous multi-hits', async ({ page }) => {
    const playback = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      audio.clearImpactHistory();
      await audio.setActive(true);

      const before = audio.played;

      const mockRun = {
        id: 'test-multihit-run',
        counter: 50,
        room: 0,
        status: 'fighting',
        build: { weapon: 'Iron Broadsword', class: 'vanguard' },
        player: { id: 'player', x: 200, y: 350, hp: 100, max_hp: 100, mana: 50, facing: 1, pose: 'attack', jump: 0 },
        enemies: [
          { id: 'g1', kind: 'goblin', x: 240, y: 350, hp: 80, max_hp: 100 },
          { id: 'g2', kind: 'goblin', x: 250, y: 350, hp: 80, max_hp: 100 },
          { id: 'g3', kind: 'goblin', x: 260, y: 350, hp: 80, max_hp: 100 },
          { id: 'g4', kind: 'goblin', x: 270, y: 350, hp: 80, max_hp: 100 },
        ],
        projectiles: [],
        drops: [],
        events: [
          // 4 simultaneous identical weapon impacts from cleave
          { id: 50, kind: 'hit_blade', x: 240, y: 350, value: 20 },
          { id: 51, kind: 'hit_blade', x: 250, y: 350, value: 20 },
          { id: 52, kind: 'hit_blade', x: 260, y: 350, value: 20 },
          { id: 53, kind: 'hit_blade', x: 270, y: 350, value: 20 },
        ]
      };

      window.RiftRenderer.snapshot(mockRun, false);
      const played = audio.played - before;
      return played;
    });

    // 4 identical blade hits in snapshot are reduced to 2
    expect(playback).toBe(2);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });
});
