// Keyboard-driver waypoints only; never changes game state or actor stats.
function createLedgeNavigator(){
 let route=null;
 return function waypoint(run,target){
  const p=run.player,edges=run.level?.rooms?.[run.room]?.drop_edges||[];
  if(!target){route=null;return null;}
  if(route){
   const edge=edges.find(edge=>edge.id===route.id);
   if(!edge||route.room!==run.room||target.y>=edge.landing_y||p.y<=edge.y-12)route=null;
   else if(Math.abs(p.x-route.x)>6)return {x:route.x,y:p.y};
   else return {x:route.x,y:edge.y-18};
  }
  for(const edge of edges){
   if(p.y>=edge.landing_y&&target.y<edge.landing_y&&p.x>=edge.x&&p.x<=edge.x+edge.w){
    const left=edge.x-24,right=edge.x+edge.w+24;
    route={id:edge.id,room:run.room,x:target.x>edge.x+edge.w?right:target.x<edge.x?left:p.x-edge.x<=edge.x+edge.w-p.x?left:right};
    return {x:route.x,y:p.y};
   }
  }
  return target;
 };
}
module.exports={createLedgeNavigator};
