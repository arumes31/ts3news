const {test,expect}=require('@playwright/test');
for(const kind of ['fire','ice','poison','thorns','rune','radiant','void','spikes'])test('authored '+kind+' hazard fixture stays paused across reload',async({page})=>{
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift?scenario=hazard-'+kind);
 await expect(page.locator('#rift-start')).toBeEnabled();
 const read=async()=>(await(await page.request.get('/api/abyss/rift')).json()).run;
 const run=await read();expect(run.paused).toBe(true);expect(run.status).toBe('fighting');
 const hazards=run.level.rooms[run.room].hazards;expect(hazards).toHaveLength(1);expect(hazards[0].kind).toBe(kind);
 const h=hazards[0],phase=(run.clock+h.offset)%h.period;
 expect(phase).toBeGreaterThanOrEqual(1.2);expect(phase).toBeLessThan(1.2+h.duration);
 expect(await page.evaluate(kind=>RiftRenderer.getHazardKindsWithPatterns().includes(kind),kind)).toBe(true);
 await page.evaluate(run=>RiftRenderer.snapshot(run),run);
 await page.evaluate(()=>history.replaceState(null,'','/abyss/rift'));await page.reload();
 await expect(page.locator('#rift-start')).toBeEnabled();
 const saved=await read();expect(saved.clock).toBe(run.clock);expect(saved.level.rooms[saved.room].hazards).toEqual(hazards);expect(saved.paused).toBe(true);
 expect(errors).toEqual([]);
});
