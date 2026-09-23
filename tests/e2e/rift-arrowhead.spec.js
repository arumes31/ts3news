const {test,expect}=require('@playwright/test');
test('legacy arrows retain a pointed outlined head when shape overlays are disabled',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(run=>{const ctx=document.getElementById('rift-canvas').getContext('2d');window.arrowheads=[];let path=[];for(const name of ['beginPath','moveTo','lineTo','stroke']){const original=ctx[name].bind(ctx);ctx[name]=(...args)=>{if(name==='beginPath')path=[];if(name==='moveTo'||name==='lineTo')path.push(args);if(name==='stroke'&&JSON.stringify(path)==='[[16,0],[6,-6],[6,6]]')window.arrowheads.push({color:ctx.strokeStyle,width:ctx.lineWidth});return original(...args);};}window.RiftDisplay.projectileShapes=false;run.projectiles=[{kind:'arrow',x:400,y:400,vx:100,vy:0,enemy:true}];window.RiftRenderer.snapshot(run,true);},run);
 await expect.poll(()=>page.evaluate(()=>window.arrowheads.length)).toBeGreaterThan(0);expect(await page.evaluate(()=>window.arrowheads.every(head=>head.color==='#14221d'&&head.width>=1))).toBe(true);
});
