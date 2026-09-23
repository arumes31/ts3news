const {test,expect}=require('@playwright/test');

test('projectile shapes use actual horizontal, vertical and diagonal velocity',async({page})=>{
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();const angles=await page.evaluate(()=>{const ctx=document.getElementById('rift-canvas').getContext('2d'),rotate=ctx.rotate.bind(ctx),angles=[];ctx.rotate=a=>{angles.push(a);rotate(a);};for(const enemy of [false,true])for(const [vx,vy] of [[100,0],[-100,0],[0,100],[0,-100],[100,100],[-100,-100],[0,0]])window.RiftRenderer.drawProjectileShape(ctx,400,300,enemy,vx,vy,'arrow');ctx.rotate=rotate;return angles;});
 const expected=[0,Math.PI,Math.PI/2,-Math.PI/2,Math.PI/4,-3*Math.PI/4,0,0,Math.PI,Math.PI/2,-Math.PI/2,Math.PI/4,-3*Math.PI/4,Math.PI];expect(angles).toHaveLength(expected.length);angles.forEach((a,i)=>expect(a).toBeCloseTo(expected[i],8));
});

test('legacy arrow sprites rotate vertically when projectile shapes are disabled',async({page})=>{
 await page.goto('/abyss/rift?scenario=checkpoint');await expect(page.locator('#rift-start')).toBeEnabled();const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.evaluate(run=>{window.arrowAngles=[];const ctx=document.getElementById('rift-canvas').getContext('2d'),fill=ctx.fillRect.bind(ctx);ctx.fillRect=(...args)=>{if(ctx.fillStyle==='#d8b3e9'){const m=ctx.getTransform();window.arrowAngles.push(Math.atan2(m.b,m.a));}return fill(...args);};window.RiftDisplay.projectileShapes=false;run.projectiles=[{kind:'arrow',x:400,y:400,vx:0,vy:100,enemy:true}];window.RiftRenderer.snapshot(run,true);},run);await expect.poll(()=>page.evaluate(()=>window.arrowAngles.length)).toBeGreaterThan(0);for(const angle of await page.evaluate(()=>window.arrowAngles))expect(angle).toBeCloseTo(Math.PI/2,8);
});
