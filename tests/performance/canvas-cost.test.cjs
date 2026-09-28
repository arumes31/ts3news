const {test}=require('node:test');const assert=require('node:assert/strict');
const {installCanvasCostProbe}=require('../../scripts/brawl-canvas-cost.cjs');
test('records operation costs by preceding image and restores original methods',()=>{
 let clock=0;const ctx={drawImage(image){clock+=2;return image;},save(){clock+=10;return 7;},restore(){clock+=1;}},old={...ctx};
 const probe=installCanvasCostProbe({context:ctx,clock:()=>clock,identify:image=>image.name});const image={name:'sprite'};
 assert.equal(ctx.drawImage(image),image);assert.equal(ctx.save(),7);ctx.restore();
 const rows=probe.stop();assert.deepEqual(ctx,old);assert.deepEqual(rows.find(r=>r.operation==='save'),{operation:'save',image:'sprite',calls:1,totalMS:10,maxMS:10,slowCalls:1});assert.deepEqual(probe.stop(),rows);
});
test('exceptions keep their identity and the wrapper records their cost',()=>{
 let clock=0;const error=new Error('sentinel'),ctx={drawImage(){clock+=4;throw error;},save(){},restore(){}};
 const probe=installCanvasCostProbe({context:ctx,clock:()=>clock,identify:()=> 'image'});assert.throws(()=>ctx.drawImage({}),e=>e===error);assert.equal(probe.stop()[0].totalMS,4);
});
