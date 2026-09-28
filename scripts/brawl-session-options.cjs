'use strict';
function sessionOptions(env){
 const smoke=env.BRAWL_SESSION_SMOKE==='1',postCap=env.BRAWL_SESSION_POST_CAP==='1';
 if(smoke&&postCap)throw Error('Smoke and post-cap modes are mutually exclusive');
 return {smoke,postCap,samples:smoke||postCap?1:3,durationMS:smoke?60000:1800000,minimumReplays:postCap?60:0,
  replayCheckpoints:postCap?[49,54,59,60]:[],timeoutMS:postCap?3600000:2400000,
  mode:smoke?'smoke (not a gate run)':postCap?'at least 60 complete missions and 30 minutes; post-history-cap diagnostic':'30 minutes of repeated complete three-tier missions'};
}
module.exports={sessionOptions};
