// Follow source images through the renderer's offscreen frame cache.
async function installAtlasProbe(page) {
  await page.addInitScript(() => {
    window.riftAtlasSources = new WeakMap();
    const draw = CanvasRenderingContext2D.prototype.drawImage;
    CanvasRenderingContext2D.prototype.drawImage = function(image, ...args) {
      const source = image.src
        ? {src: image.src, height: image.height, y: args.length === 8 ? args[1] : 0}
        : window.riftAtlasSources.get(image);
      if (source) window.riftAtlasSources.set(this.canvas, source);
      return draw.call(this, image, ...args);
    };
  });
}
module.exports = {installAtlasProbe};
