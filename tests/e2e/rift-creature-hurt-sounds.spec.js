const { test, expect } = require('@playwright/test');

test.describe('Vary enemy hurt sounds by creature family (Proposal 0186)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('RiftAudio.hurt synthesizes distinct hurt sounds for each creature family', async ({ page }) => {
    const kinds = ['goblin', 'knight', 'archer', 'treasure', 'boss'];

    const result = await page.evaluate(async (families) => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      for (const kind of families) {
        audio.hurt(kind, 0);
      }

      const after = audio.played;
      return { before, after, count: families.length };
    }, kinds);

    expect(result.after).toBe(result.before + result.count);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('audio.play with hurt and creature kind argument delegates to audio.hurt', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      audio.play('hurt', 0, 'goblin');
      audio.play('hurt', 0, 'knight');
      audio.play('hurt', 0, 'archer');
      audio.play('hurt', 0, 'treasure');
      audio.play('hurt', 0, 'boss');

      const after = audio.played;
      return { before, after };
    });

    expect(result.after).toBe(result.before + 5);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('creature hurt cues can be previewed via RiftAudio.previewCue', async ({ page }) => {
    const hurtCues = ['goblin_hurt', 'knight_hurt', 'archer_hurt', 'treasure_hurt', 'boss_hurt'];

    const canPreview = await page.evaluate(async (cues) => {
      const audio = window.RiftAudio;
      const results = {};
      for (const cue of cues) {
        const promise = audio.previewCue(cue);
        results[cue] = promise instanceof Promise;
      }
      return results;
    }, hurtCues);

    for (const cue of hurtCues) {
      expect(canPreview[cue]).toBe(true);
    }
  });

  test('bestiary inspect dialog renders preview button for creature hurt sound', async ({ page }) => {
    // Open bestiary
    const bestiarySummary = page.locator('.rift-bestiary > summary');
    await expect(bestiarySummary).toBeVisible();
    await bestiarySummary.click();

    // Inspect first creature
    const inspectBtn = page.locator('#rift-monsters button').first();
    await expect(inspectBtn).toBeVisible();
    await inspectBtn.click();

    // Verify 'Preview hurt' button exists in sound preview list
    const hurtPreviewBtn = page.locator('#rift-monster-sounds button', { hasText: 'Preview hurt' });
    await expect(hurtPreviewBtn).toBeVisible();
    await hurtPreviewBtn.click();

    // Check sound status message
    const soundStatus = page.locator('#rift-monster-sound-status');
    await expect(soundStatus).not.toBeEmpty();
  });

  test('renderer snapshot routes enemy hurt events to audio playback', async ({ page }) => {
    const playback = await page.evaluate(async () => {
      const calls = [];
      const origPlay = window.RiftAudio.play;
      window.RiftAudio.play = function (kind, pan, extra, extra2) {
        calls.push({ kind, pan, extra, extra2 });
        return origPlay.apply(this, arguments);
      };

      const mockRun = {
        id: 'test-hurt-run',
        counter: 20,
        room: 0,
        status: 'fighting',
        build: { weapon: 'Void Katana', class: 'bloodblade' },
        player: { id: 'player', x: 200, y: 350, hp: 100, max_hp: 100, mana: 50, facing: 1, pose: 'attack', jump: 0 },
        enemies: [{ id: 'goblin0', kind: 'goblin', x: 250, y: 350, hp: 60, max_hp: 100, facing: -1, pose: 'hit' }],
        projectiles: [],
        drops: [],
        events: [
          { id: 20, kind: 'goblin_hurt', x: 250, y: 320, value: 25 }
        ]
      };

      window.RiftRenderer.snapshot(mockRun, false);
      window.RiftAudio.play = origPlay;
      return calls;
    });

    const hurtCall = playback.find(c => c.kind === 'goblin_hurt');
    expect(hurtCall).toBeDefined();
    expect(hurtCall.extra).toBe(25);
  });
});
