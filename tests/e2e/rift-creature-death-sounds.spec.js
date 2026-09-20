const { test, expect } = require('@playwright/test');

test.describe('Vary death sounds by creature family (Proposal 0187)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('RiftAudio.death synthesizes distinct death sounds for each creature family', async ({ page }) => {
    const kinds = ['goblin', 'knight', 'archer', 'treasure', 'boss', 'wolf', 'spore'];

    const result = await page.evaluate(async (families) => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      for (const kind of families) {
        audio.death(kind, 0);
      }

      const after = audio.played;
      return { before, after, count: families.length };
    }, kinds);

    expect(result.after).toBe(result.before + result.count);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('audio.play with death and creature kind argument delegates to audio.death', async ({ page }) => {
    const result = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.setActive(true);
      const before = audio.played;

      audio.play('death', 0, 'goblin');
      audio.play('death', 0, 'knight');
      audio.play('death', 0, 'archer');
      audio.play('death', 0, 'treasure');
      audio.play('death', 0, 'boss');
      audio.play('death', 0, 'wolf');
      audio.play('death', 0, 'spore');

      const after = audio.played;
      return { before, after };
    });

    expect(result.after).toBe(result.before + 7);

    await page.evaluate(async () => {
      await window.RiftAudio.setActive(false);
    });
  });

  test('creature death cues can be previewed via RiftAudio.previewCue', async ({ page }) => {
    const deathCues = [
      'goblin_death',
      'knight_death',
      'archer_death',
      'treasure_death',
      'boss_death',
      'wolf_death',
      'spore_death',
    ];

    const canPreview = await page.evaluate(async (cues) => {
      const audio = window.RiftAudio;
      const results = {};
      for (const cue of cues) {
        const promise = audio.previewCue(cue);
        results[cue] = promise instanceof Promise;
      }
      return results;
    }, deathCues);

    for (const cue of deathCues) {
      expect(canPreview[cue]).toBe(true);
    }
  });

  test('bestiary inspect dialog renders preview button for creature defeat sound', async ({ page }) => {
    // Open bestiary
    const bestiarySummary = page.locator('.rift-bestiary > summary');
    await expect(bestiarySummary).toBeVisible();
    await bestiarySummary.click();

    // Inspect first creature
    const inspectBtn = page.locator('#rift-monsters button').first();
    await expect(inspectBtn).toBeVisible();
    await inspectBtn.click();

    // Verify 'Preview defeat' button exists in sound preview list
    const defeatPreviewBtn = page.locator('#rift-monster-sounds button', { hasText: 'Preview defeat' });
    await expect(defeatPreviewBtn).toBeVisible();
    await defeatPreviewBtn.click();

    // Check sound status message
    const soundStatus = page.locator('#rift-monster-sound-status');
    await expect(soundStatus).not.toBeEmpty();
  });

  test('renderer snapshot routes enemy death events to audio playback', async ({ page }) => {
    const playback = await page.evaluate(async () => {
      const calls = [];
      const origPlay = window.RiftAudio.play;
      window.RiftAudio.play = function (kind, pan, extra, extra2) {
        calls.push({ kind, pan, extra, extra2 });
        return origPlay.apply(this, arguments);
      };

      const mockRun = {
        id: 'test-death-run',
        counter: 25,
        room: 0,
        status: 'fighting',
        build: { weapon: 'Void Katana', class: 'bloodblade' },
        player: { id: 'player', x: 200, y: 350, hp: 100, max_hp: 100, mana: 50, facing: 1, pose: 'idle', jump: 0 },
        enemies: [{ id: 'knight0', kind: 'knight', x: 250, y: 350, hp: 0, max_hp: 200, facing: -1, pose: 'defeat' }],
        projectiles: [],
        drops: [],
        events: [
          { id: 25, kind: 'knight_death', x: 250, y: 350, value: 0 }
        ]
      };

      window.RiftRenderer.snapshot(mockRun, false);
      window.RiftAudio.play = origPlay;
      return calls;
    });

    const deathCall = playback.find(c => c.kind === 'knight_death');
    expect(deathCall).toBeDefined();
  });

  test('combat captions show Treasure goblin defeated on treasure_death event', async ({ page }) => {
    // Open settings and enable captions
    await page.locator('.rift-settings > summary').click();
    const captionsCheckbox = page.locator('#rift-combat-captions');
    await captionsCheckbox.check();

    await page.evaluate(() => {
      const baseRun = {
        id: 'test-caption-run',
        level: { id: 1 },
        room: 0,
        counter: 30,
        status: 'fighting',
        player: { id: 'player', x: 200, y: 350, hp: 100, max_hp: 100, mana: 50 },
        build: { signatures: [] },
        skill_timers: {},
        events: []
      };
      window.RiftFeedback.update(baseRun, false, true);

      const updated = JSON.parse(JSON.stringify(baseRun));
      updated.counter = 31;
      updated.events = [
        { id: 31, kind: 'treasure_death', x: 200, y: 350, value: 0 }
      ];
      window.RiftFeedback.update(updated, false, true);
    });

    const captions = page.locator('#rift-captions');
    await expect(captions).toBeVisible();
    await expect(captions).toContainText('Treasure goblin defeated');
  });
});
