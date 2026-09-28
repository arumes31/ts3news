(function(global){
  'use strict';
  function create({manifest,decode}){
    if(manifest?.version!==1||Object.keys(manifest.sources||{}).length!==4||Object.keys(manifest.rigs||{}).length!==32||typeof decode!=='function'){
      throw new Error('Creature artwork manifest is unavailable. Reload to try again.');
    }
    for(const source of Object.values(manifest.sources)){
      if(source.width!==1254||source.height!==1254)throw new Error('Creature artwork manifest dimensions differ.');
    }
    for(const panel of Object.values(manifest.rigs)){
      if(!Object.hasOwn(manifest.sources,panel.sourceAsset)||panel.width!==1254||!Number.isInteger(panel.top)||panel.top<0||!Number.isInteger(panel.height)||panel.height<=0||panel.top+panel.height>1254||typeof panel.url!=='string'){
        throw new Error('Creature artwork manifest row differs.');
      }
    }
    const surfaces=new Map(),prepared=new Set(),pending=new Map(),retries=new Map();
    let disposed=false;
    const closed=()=>new Error('Creature artwork loader is closed.');
    function prepareFrame(frame){
      if(!frame?.source)return Promise.resolve(); // Legacy fallbacks use the local actor atlas.
      const panel=manifest.rigs[frame.rig];
      if(prepared.has(frame.rig))return Promise.resolve();
      if(!pending.has(frame.rig)){
        const attempt=retries.get(frame.rig)||0,url=panel.url+(attempt?(panel.url.includes('?')?'&':'?')+'retry='+attempt:'');
        const load=Promise.resolve().then(()=>decode(url)).then(source=>{
          try{
            if(disposed)throw closed();
            if(source.width!==panel.width||source.height!==panel.height)throw new Error('Creature artwork dimensions differ.');
            let surface=surfaces.get(panel.sourceAsset);
            if(!surface){
              const canvas=document.createElement('canvas');canvas.width=1254;canvas.height=1254;
              const context=canvas.getContext('2d');
              if(!context){canvas.width=0;canvas.height=0;throw new Error('Creature artwork canvas is unavailable.');}
              surface={canvas,context};surfaces.set(panel.sourceAsset,surface);
            }
            // Preserve original fractional sampling. Clear shared borders before copying.
            surface.context.imageSmoothingEnabled=false;
            surface.context.clearRect(0,panel.top,panel.width,panel.height);
            surface.context.drawImage(source,0,panel.top);
            prepared.add(frame.rig);
          }finally{
            if(typeof source?.close==='function')source.close();
          }
        }).catch(error=>{
          if(disposed)throw closed();
          retries.set(frame.rig,attempt+1);
          throw new Error('Could not load creature artwork. Retry to continue.',{cause:error});
        }).finally(()=>pending.delete(frame.rig));
        pending.set(frame.rig,load);
      }
      return pending.get(frame.rig);
    }
    return Object.freeze({
      async prepare(frames){
        if(disposed)throw closed();
        if(!Array.isArray(frames))throw new TypeError('Creature frames must be a list.');
        // Validate the complete request before starting any asynchronous work.
        for(const frame of frames){
          if(!frame?.source)continue; // Legacy fallbacks use the local actor atlas.
          if(!Object.hasOwn(manifest.rigs,frame.rig)||manifest.rigs[frame.rig].sourceAsset!==frame.asset){
            throw new Error('Creature artwork manifest does not match this frame. Reload to try again.');
          }
        }
        await Promise.all(frames.map(prepareFrame));
      },
      image(frame){
        if(disposed||!prepared.has(frame?.rig)||!Object.hasOwn(manifest.rigs,frame.rig)||manifest.rigs[frame.rig].sourceAsset!==frame.asset)return null;
        return surfaces.get(frame.asset)?.canvas||null;
      },
      stats(){return {surfaces:surfaces.size,preparedRows:prepared.size,pendingRows:pending.size,surfaceBytes:surfaces.size*1254*1254*4};},
      dispose(){
        disposed=true;
        for(const {canvas} of surfaces.values()){canvas.width=0;canvas.height=0;}
        surfaces.clear();prepared.clear();retries.clear();
      }
    });
  }
  global.RiftCreatureLoader=Object.freeze({create});
})(window);
