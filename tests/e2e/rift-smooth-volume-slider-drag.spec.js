const { test, expect } = require('@playwright/test');

test.describe('Avoid abrupt gain changes when dragging volume sliders (Proposal 0199)', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/abyss/rift');
    await page.evaluate(() => {
      window.RiftAudio.set('music', 0.5, false);
      window.RiftAudio.set('effects', 0.65, false);
    });
    await expect(page.locator('#rift-start')).toBeEnabled();
  });

  test('audio.set with isDragging=true sets dragging state and debounces localStorage saves', async ({ page }) => {
    const results = await page.evaluate(async () => {
      const audio = window.RiftAudio;
      await audio.unlock();

      // Clear any prior stored value
      localStorage.removeItem('riftAudio:music');

      // Drag slider: rapid intermediate values with isDragging=true
      audio.set('music', 0.40, true);
      const drag1 = {
        isDragging: audio.isDragging,
        activeDragKey: audio.activeDragKey,
        isDraggingSlider: audio.isDraggingSlider('music'),
        storedImmediate: localStorage.getItem('riftAudio:music'),
      };

      audio.set('music', 0.30, true);
      audio.set('music', 0.20, true);
      const drag2 = {
        isDragging: audio.isDragging,
        activeDragKey: audio.activeDragKey,
        storedImmediate: localStorage.getItem('riftAudio:music'),
      };

      // Release drag: isDragging=false
      audio.set('music', 0.20, false);
      const dragEnd = {
        isDragging: audio.isDragging,
        activeDragKey: audio.activeDragKey,
        storedFinal: localStorage.getItem('riftAudio:music'),
      };

      return { drag1, drag2, dragEnd };
    });

    expect(results.drag1.isDragging).toBe(true);
    expect(results.drag1.activeDragKey).toBe('music');
    expect(results.drag1.isDraggingSlider).toBe(true);
    // localStorage was debounced, not immediately written
    expect(results.drag1.storedImmediate).toBeNull();

    expect(results.drag2.isDragging).toBe(true);
    expect(results.drag2.storedImmediate).toBeNull();

    expect(results.dragEnd.isDragging).toBe(false);
    expect(results.dragEnd.activeDragKey).toBeNull();
    // On release (isDragging=false), final value was flushed to localStorage
    expect(results.dragEnd.storedFinal).toBe('0.2');
  });

  test('UI range sliders dispatch drag events on input and commit on change and pointerup', async ({ page }) => {
    const details = page.locator('.rift-settings');
    await details.evaluate(node => { node.open = true; });

    const musicSlider = page.locator('#rift-music-volume');
    await expect(musicSlider).toBeVisible();

    // Verify slider drag interaction
    const stateDuringAndAfter = await page.evaluate(async () => {
      const slider = document.getElementById('rift-music-volume');
      const audio = window.RiftAudio;

      // Simulate pointer drag by firing input
      slider.value = '75';
      slider.dispatchEvent(new Event('input', { bubbles: true }));

      const duringDrag = {
        isDragging: audio.isDragging,
        activeDragKey: audio.activeDragKey,
        currentMusic: audio.music,
      };

      // Simulate pointer release by firing change
      slider.dispatchEvent(new Event('change', { bubbles: true }));

      const afterDrag = {
        isDragging: audio.isDragging,
        activeDragKey: audio.activeDragKey,
        currentMusic: audio.music,
        stored: localStorage.getItem('riftAudio:music'),
      };

      return { duringDrag, afterDrag };
    });

    expect(stateDuringAndAfter.duringDrag.isDragging).toBe(true);
    expect(stateDuringAndAfter.duringDrag.activeDragKey).toBe('music');
    expect(stateDuringAndAfter.duringDrag.currentMusic).toBe(0.75);

    expect(stateDuringAndAfter.afterDrag.isDragging).toBe(false);
    expect(stateDuringAndAfter.afterDrag.activeDragKey).toBeNull();
    expect(stateDuringAndAfter.afterDrag.currentMusic).toBe(0.75);
    expect(stateDuringAndAfter.afterDrag.stored).toBe('0.75');
  });

  test('effects volume slider in HUD settings supports smooth dragging', async ({ page }) => {
    const details = page.locator('.rift-settings');
    await details.evaluate(node => { node.open = true; });

    const effectsSlider = page.locator('#rift-effects-volume');
    await expect(effectsSlider).toBeVisible();

    const state = await page.evaluate(() => {
      const slider = document.getElementById('rift-effects-volume');
      const audio = window.RiftAudio;

      slider.value = '25';
      slider.dispatchEvent(new Event('input', { bubbles: true }));
      const during = { isDragging: audio.isDragging, key: audio.activeDragKey, val: audio.effects };

      slider.dispatchEvent(new Event('pointerup', { bubbles: true }));
      const after = { isDragging: audio.isDragging, key: audio.activeDragKey, val: audio.effects };

      return { during, after };
    });

    expect(state.during.isDragging).toBe(true);
    expect(state.during.key).toBe('effects');
    expect(state.during.val).toBe(0.25);

    expect(state.after.isDragging).toBe(false);
    expect(state.after.key).toBeNull();
    expect(state.after.val).toBe(0.25);
  });
});
