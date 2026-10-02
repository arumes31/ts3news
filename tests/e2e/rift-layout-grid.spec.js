const {test,expect}=require('@playwright/test');
test('optional world grid persists, follows the camera and hides for clean screenshots',async({page})=>{
 await page.addInitScript(()=>{window.gridLabels=[];const fill=CanvasRenderingContext2D.prototype.fillText;CanvasRenderingContext2D.prototype.fillText=function(text,x,y,...args){if(String(text).startsWith('GRID 50')||/^X \d+$/.test(String(text)))window.gridLabels.push({text,x,y});if(window.gridLabels.length>400)window.gridLabels.splice(0,200);return fill.call(this,text,x,y,...args);};});
 await page.goto('/abyss/rift?scenario=drop-edge');await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('#rift-auto').uncheck();await page.locator('#rift-start').click();await expect(page.locator('#rift-overlay')).toBeHidden();
 await page.locator('.rift-settings > summary').click();const grid=page.getByLabel('Arena layout grid (50-unit world coordinates)',{exact:true});await expect(grid).not.toBeChecked();await grid.check();
 await expect.poll(()=>page.evaluate(()=>gridLabels.some(l=>l.text.startsWith('GRID 50')))).toBe(true);
 await page.locator('.rift-settings > summary').click();await page.locator('#rift-viewport').screenshot({path:'test-results/layout-grid.png'});
 const before=await page.evaluate(()=>gridLabels.filter(l=>l.text==='X 500').at(-1).x);
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 await page.keyboard.down('KeyD');try{await expect.poll(async()=>(await saved()).player.x,{intervals:[20]}).toBeGreaterThan(650);}finally{await page.keyboard.up('KeyD');}
 await expect.poll(()=>page.evaluate(()=>gridLabels.filter(l=>l.text==='X 500').at(-1).x)).toBeLessThan(before-180);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();await page.locator('.rift-settings > summary').click();await expect(grid).toBeChecked();
 await page.getByRole('checkbox',{name:'Clean screenshot mode (hide HUD)',exact:true}).check();await page.evaluate(()=>gridLabels=[]);await page.waitForTimeout(200);expect(await page.evaluate(()=>gridLabels.length)).toBe(0);
 await page.getByRole('checkbox',{name:'Clean screenshot mode (hide HUD)',exact:true}).uncheck();await expect.poll(()=>page.evaluate(()=>gridLabels.length)).toBeGreaterThan(0);
 await page.locator('#rift-reset-display').click();await expect(grid).not.toBeChecked();expect(await page.evaluate(()=>JSON.parse(localStorage.getItem('riftDisplay')).layoutGrid)).toBe(false);
 await page.setViewportSize({width:390,height:844});expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
