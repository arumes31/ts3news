const {test,expect}=require('@playwright/test');
test('guardian arrival stays clear of active hazards and survives reload unchanged',async({page})=>{
 await page.goto('/abyss/rift?scenario=spawn-hazards');await expect(page.locator('#rift-start')).toBeEnabled();
 const saved=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const before=await saved();expect(before.room).toBe(2);expect(before.enemies.some(e=>e.kind==='boss')).toBe(true);
 const hazards=before.level.rooms[2].hazards;expect(hazards.some(h=>{const phase=(before.clock+h.offset)%h.period;return phase>=1.2&&phase<1.2+h.duration;})).toBe(true);
 for(const enemy of before.enemies){const radius=enemy.kind==='boss'?32:['wolf','treasure'].includes(enemy.kind)?6:10;for(const h of hazards)expect(enemy.x>h.x-radius&&enemy.x<h.x+h.w+radius&&enemy.y>h.y-radius&&enemy.y<h.y+h.h+radius).toBe(false);}
 for(const boss of before.enemies.filter(e=>e.kind==='boss')){expect(boss.x).toBeGreaterThanOrEqual(49);expect(boss.x).toBeLessThanOrEqual(1551);expect(boss.y).toBeGreaterThanOrEqual(329);expect(boss.y).toBeLessThanOrEqual(476);for(const other of before.enemies.filter(e=>e.id!==boss.id&&e.hp>0)){const radius=other.kind==='boss'?18:['wolf','treasure'].includes(other.kind)?6:10;expect(Math.abs(other.x-boss.x)<60+radius&&Math.abs(other.y-boss.y)<32+radius).toBe(false);}}
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();await expect(page.locator('#rift-start')).toBeEnabled();expect((await saved()).enemies).toEqual(before.enemies);
});
