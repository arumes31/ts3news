"""Inspect reachable payload shapes, direct run retainers and root paths in local restart or session fixture heaps.

These counts are diagnostic evidence, not a dominator retained-size calculation
or an automatic leak verdict. Use only synthetic benchmark artifacts.
"""
import argparse
from array import array
from collections import deque
import json
from pathlib import Path
def root_paths(heap, targets):
 """One shortest non-weak snapshot-graph path, not a dominator/leak verdict."""
 meta=heap['snapshot']['meta'];nf=meta['node_fields'];ef=meta['edge_fields']
 nodes=heap['nodes'];edges=heap['edges'];strings=heap['strings'];ns=len(nf);es=len(ef)
 ti=nf.index('type');ni=nf.index('name');ec=nf.index('edge_count')
 et=ef.index('type');en=ef.index('name_or_index');to=ef.index('to_node')
 types=meta['node_types'][ti];edge_types=meta['edge_types'][et];count=len(nodes)//ns
 if not count or types[nodes[ti]]!='synthetic':raise ValueError('Expected a synthetic snapshot root')
 offsets=array('Q',[0]);offset=0
 for i in range(0,len(nodes),ns):
  offset+=nodes[i+ec]*es;offsets.append(offset)
 parents=array('q',[-1])*count;parent_edges=array('q',[-1])*count
 parents[0]=0;queue=deque([0]);remaining={i//ns for i in targets}
 while queue and remaining:
  owner=queue.popleft();remaining.discard(owner)
  for j in range(offsets[owner],offsets[owner+1],es):
   if edge_types[edges[j+et]]=='weak':continue
   child=edges[j+to]//ns
   if parents[child]!=-1:continue
   parents[child]=owner;parent_edges[child]=j;queue.append(child)
 results=[]
 for target in sorted(targets):
  current=target//ns;path=[]
  if parents[current]==-1:
   results.append({'targetIndex':current,'status':'no non-weak root path','path':[]});continue
  while len(path)<128:
   step={'index':current,'type':types[nodes[current*ns+ti]],'name':strings[nodes[current*ns+ni]]}
   if current:
    j=parent_edges[current];kind=edge_types[edges[j+et]]
    step['via']={'edgeType':kind,'edge':edges[j+en] if kind in ('element','hidden') else strings[edges[j+en]]}
   path.append(step)
   if current==0:break
   current=parents[current]
  results.append({'targetIndex':target//ns,'status':'found' if path[-1]['index']==0 else 'truncated at 128 nodes','path':list(reversed(path))})
 return results

parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('directory',type=Path,help='Benchmark output directory containing per-sample memory-report.json files')
root=parser.parse_args().directory
results=[]
for report_path in sorted(root.glob('*/memory-report.json')):
 report=json.loads(report_path.read_text(encoding='utf-8'))
 sample={'sample':report['sample'],'checkpoints':[]}
 for point in report['checkpoints']:
  if 'reachable' not in point:continue
  checkpoint_key='mission' if 'mission' in point else 'cycle'
  h=json.loads(Path(point['snapshot']).read_text(encoding='utf-8'));m=h['snapshot']['meta'];nf=m['node_fields'];ef=m['edge_fields'];nodes=h['nodes'];edges=h['edges'];strings=h['strings'];ns=len(nf);es=len(ef)
  nt=m['node_types'][nf.index('type')];et=m['edge_types'][ef.index('type')]
  ti=nf.index('type');ni=nf.index('name');ei=nf.index('edge_count');eti=ef.index('type');eni=ef.index('name_or_index');eto=ef.index('to_node')
  network_nodes=[];detached_nodes=[];attempt_nodes=[];network_bytes=0
  counts={'actorShapes':0,'effectShapes':0,'runShapes':0,'attemptRecordShapes':0};names={};run_nodes=set();run_ids=[];off=0
  for i in range(0,len(nodes),ns):
   typ=nt[nodes[i+ti]];name=strings[nodes[i+ni]];count=nodes[i+ei]
   if typ=='native' and name=='blink::NetworkResourcesData::ResourceData':
    network_nodes.append(i)
    if 'self_size' in nf:network_bytes+=nodes[i+nf.index('self_size')]
   if 'detachedness' in nf and nodes[i+nf.index('detachedness')]==2:detached_nodes.append(i)
   if typ=='object':
    props={strings[edges[j+eni]] for j in range(off,off+count*es,es) if et[edges[j+eti]]=='property'}
    if {'hp','max_hp','kind'}.issubset(props):counts['actorShapes']+=1
    if {'x','y','started','kind'}.issubset(props):counts['effectShapes']+=1
    if {'mission','outcome','at_ms','splits','hp','class'}.issubset(props):
     counts['attemptRecordShapes']+=1;attempt_nodes.append(i)
    if {'player','enemies','status'}.issubset(props):
     counts['runShapes']+=1;run_nodes.add(i)
     for j in range(off,off+count*es,es):
      if et[edges[j+eti]]=='property' and strings[edges[j+eni]]=='id':run_ids.append(strings[nodes[edges[j+eto]+ni]])
   if typ in ('native','object') and name in ['AudioBuffer','AudioBufferSourceNode','OscillatorNode','GainNode','EventListener','blink::DOMTimer','blink::(anonymous namespace)::DOMTimerCoordinator']:names[name]=names.get(name,0)+1
   off+=count*es
  retainers=[];off=0
  for i in range(0,len(nodes),ns):
   count=nodes[i+ei]
   for j in range(off,off+count*es,es):
    if edges[j+eto] in run_nodes and et[edges[j+eti]]!='weak':
     kind=et[edges[j+eti]];edge=edges[j+eni] if kind in ('element','hidden') else strings[edges[j+eni]]
     retainers.append({'targetIndex':edges[j+eto]//ns,'ownerType':nt[nodes[i+ti]],'ownerName':strings[nodes[i+ni]],'edgeType':kind,'edge':edge})
   off+=count*es
  # Native samples are positional examples, not proof of every resource owner.
  network_examples={network_nodes[k] for k in (0,len(network_nodes)//2,len(network_nodes)-1)} if network_nodes else set()
  paths={p['targetIndex']*ns:p for p in root_paths(h,set(run_nodes)|network_examples|set(detached_nodes)|set(attempt_nodes))}
  selected_paths=lambda targets:[paths[i] for i in sorted(targets)]
  diagnostics={'attemptRootPaths':selected_paths(attempt_nodes),'networkResources':{'count':len(network_nodes),'selfBytes':network_bytes,'examplePaths':selected_paths(network_examples)},'detachedNodes':{'count':len(detached_nodes),'examplePaths':selected_paths(detached_nodes)}}
  sample['checkpoints'].append({**diagnostics,checkpoint_key:point[checkpoint_key],'elapsedMS':point.get('elapsedMS'),'shapes':counts,'audioAndTimers':names,'runRetainers':retainers,'runRootPaths':selected_paths(run_nodes),'retainedRunIDs':run_ids,'detached':point['reachable']['detached'],'dom':point['dom'],'usedSize':point['heap']['usedSize']})
 runs={entry['index']:entry['id'] for entry in report['expeditions']}
 for point in sample['checkpoints']:
  checkpoint=point.get('mission',point.get('cycle'))
  expected=runs['warmup' if checkpoint==0 else checkpoint]
  if set(point['retainedRunIDs'])!={expected}:
   raise ValueError('Retained run IDs differ from the current expedition; inspect this snapshot')
 results.append(sample)
if not results:raise ValueError('No benchmark reports found')
out=root/'retention-review.json';out.write_text(json.dumps(results,indent=2,ensure_ascii=True)+'\n',encoding='utf-8')
print(json.dumps([{'sample':r['sample'],'points':[{'checkpoint':p.get('mission',p.get('cycle')),'elapsedMS':p.get('elapsedMS'),'shapes':p['shapes'],'dom':p['dom'],'audio':p['audioAndTimers']} for p in r['checkpoints']]} for r in results],indent=2,ensure_ascii=True))
