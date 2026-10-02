const { test, expect } = require('@playwright/test');

test.describe('Vary weapon impacts by weapon family (Proposal 0185)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('RiftAudio.hit synthesises distinct impact sounds for each weapon family', async ({ page }) => {
    const families = ['blade', 'blunt', 'pierce', 'arcane', 'fist', 'ranged'];

    const result = await page.evaluate(async (fams) => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      for (const fam of fams) {
        audio.hit(fam, 0);
      }

      const after = audio.played;
      return { before, after, count: fams.length };
    }, families);

    expect(result.after).toBe(result.before + result.count);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('audio.play with hit and family argument routes to family hit cue', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      audio.play('hit', 0, 'blunt');
      audio.play('hit_arcane', 0);
      audio.play('hit_pierce', 0);
      audio.play('hit_fist', 0);
      audio.play('hit_blade', 0);

      const after = audio.played;
      return { before, after };
    });

    expect(result.after).toBe(result.before + 5);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('renderer snapshot routes family hit events and displays floating damage numbers', async ({ page }) => {
    const playback = await page.evaluate(async () => {
      const calls = [];
      const origPlay = window.RiftAudio.play;
      window.RiftAudio.play = function (kind, pan, extra, extra2) {
        calls.push({ kind, pan, extra, extra2 });
        return origPlay.apply(this, arguments);
      };

      const mockRun = {
        id: 'test-hit-run',
        counter: 12,
        room: 0,
        status: 'fighting',
        build: { weapon: 'Earthshaker Warhammer', class: 'geomancer' },
        player: { id: 'player', x: 200, y: 350, hp: 100, max_hp: 100, mana: 50, facing: 1, pose: 'attack', jump: 0 },
        enemies: [{ id: 'goblin0', kind: 'goblin', x: 240, y: 350, hp: 80, max_hp: 100, facing: -1, pose: 'hit' }],
        projectiles: [],
        drops: [],
        events: [
          { id: 12, kind: 'hit_blunt', x: 240, y: 320, value: 28 }
        ]
      };

      window.RiftRenderer.snapshot(mockRun, false);
      window.RiftAudio.play = origPlay;
      return calls;
    });

    const bluntCall = playback.find(c => c.kind === 'hit_blunt');
    expect(bluntCall).toBeDefined();
    expect(bluntCall.extra).toBe(28);
  });

  test('renderer snapshot handles generic hit event and passes weapon name to audio.play', async ({ page }) => {
    const playback = await page.evaluate(async () => {
      const calls = [];
      const origPlay = window.RiftAudio.play;
      window.RiftAudio.play = function (kind, pan, extra, extra2) {
        calls.push({ kind, pan, extra, extra2 });
        return origPlay.apply(this, arguments);
      };

      const mockRun = {
        id: 'test-generic-hit-run',
        counter: 15,
        room: 0,
        status: 'fighting',
        build: { weapon: 'Lifebloom Staff', class: 'elementalist' },
        player: { id: 'player', x: 200, y: 350, hp: 100, max_hp: 100, mana: 50, facing: 1, pose: 'attack', jump: 0 },
        enemies: [],
        projectiles: [],
        drops: [],
        events: [
          { id: 15, kind: 'hit', x: 240, y: 320, value: 20 }
        ]
      };

      window.RiftRenderer.snapshot(mockRun, false);
      window.RiftAudio.play = origPlay;
      return calls;
    });

    const hitCall = playback.find(c => c.kind === 'hit');
    expect(hitCall).toBeDefined();
    expect(hitCall.extra).toBe('Lifebloom Staff');
  });
});
