const {test,expect}=require('@playwright/test');
const objectiveFiles=['sigil','totem','relic','generator','spirit','cage','lantern'];

test('idle startup skips objective artwork and a whole mission is prepared together',async({page})=>{
 const requests=[];page.on('request',r=>{const match=new URL(r.url()).pathname.match(/\/rift_(sigil|totem|relic|generator|spirit|cage|lantern)\.png$/);if(match)requests.push(match[1]);});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();expect(requests).toEqual([]);
 await page.evaluate(()=>RiftRenderer.prepareRun({room:0,level:{region:0,rooms:[{objective:'carry_relic'},{objective:'rescue_companions'},{objective:'split_defense'}]}}));
 expect(requests.sort()).toEqual(['cage','lantern','relic','spirit']);
 await page.evaluate(()=>RiftRenderer.prepareRun({room:2,level:{region:0,rooms:[{objective:'carry_relic'},{objective:'rescue_companions'},{objective:'split_defense'}]}}));
 expect(requests).toHaveLength(4);
});

test('saved actors and nested objectives load without a campaign level',async({page})=>{
 const requests=[];page.on('request',r=>{const match=new URL(r.url()).pathname.match(/\/rift_(sigil|totem|relic|generator|spirit|cage|lantern)\.png$/);if(match)requests.push(match[1]);});
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 await page.evaluate(()=>RiftRenderer.prepareRun({room:0,enemies:[{kind:'totem'},{kind:'generator'},{kind:'cage'}],room_objective:{kind:'rune_gate',lantern:{kind:'lantern'},relic:{x:1,y:1}}}));
 expect(requests.sort()).toEqual(objectiveFiles.slice().sort());
});

test('required objective decode gates a saved run and retry coalesces failed loads',async({page})=>{
 let release;const held=new Promise(resolve=>release=resolve);let attempts=0;
 await page.addInitScript(()=>{
  const decode=HTMLImageElement.prototype.decode;
  window.holdTotem=new Promise(resolve=>window.releaseTotem=resolve);
  HTMLImageElement.prototype.decode=async function(){await decode.call(this);if(this.src.includes('rift_totem.png')){window.totemDecoding=true;await window.holdTotem;}};
 });
 await page.goto('/abyss/rift?scenario=totems',{waitUntil:'domcontentloaded'});
 await expect.poll(()=>page.evaluate(()=>!!window.totemDecoding)).toBe(true);await expect(page.locator('#rift-start')).toBeDisabled();
 await page.evaluate(()=>window.releaseTotem());await expect(page.locator('#rift-start')).toBeEnabled();
 await page.route('**/static/rift_cage.png*',async route=>{attempts++;if(attempts===1)await route.abort();else{await held;await route.continue();}});
 const failure=await page.evaluate(()=>RiftRenderer.prepareRun({room:0,room_objective:{kind:'rescue_companions'}}).then(()=>'',e=>e.message));
 expect(failure).toContain('Could not load cage artwork');
 const retry=page.evaluate(()=>Promise.all([RiftRenderer.prepareRun({room:0,room_objective:{kind:'rescue_companions'}}),RiftRenderer.prepareRun({room:0,room_objective:{kind:'rescue_companions'}})]));
 await expect.poll(()=>attempts).toBe(2);release();await retry;expect(attempts).toBe(2);
});
