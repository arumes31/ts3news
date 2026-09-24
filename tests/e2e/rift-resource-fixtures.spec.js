const {test,expect}=require('@playwright/test');
for(const base of ['warrior','ranger','arcanist','warden','reaver','artificer'])test(base+' resource fixtures cover foundation and subclasses at every charge',async({page})=>{
 test.setTimeout(120000);
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.goto('/abyss/rift');await expect(page.locator('#rift-start')).toBeEnabled();
 const catalog=await(await page.request.get('/api/abyss/rift')).json();
 const cls=catalog.class_options.find(c=>c.id===base);expect(cls).toBeDefined();
 for(const style of [base,...cls.subclasses.map(s=>s.id)])for(let charges=0;charges<=3;charges++){
  await page.goto('/abyss/rift?scenario=checkpoint&subclass='+style+'&charges='+charges);
  await expect(page.locator('#rift-start')).toBeEnabled();
  const run=(await(await page.request.get('/api/abyss/rift')).json()).run;
  expect(run.build.class).toBe(style);expect(run.resource||0).toBe(charges);
  await expect(page.locator('#rift-resource')).toHaveText(run.build.resource+' '+charges+'/3');
  expect(run.build.signatures).toHaveLength(2);
 }
 expect(errors).toEqual([]);
});
