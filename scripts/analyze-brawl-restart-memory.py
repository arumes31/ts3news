"""Inspect reachable payload shapes and direct run retainers in local restart or session fixture heaps.

These counts are diagnostic evidence, not a dominator retained-size calculation
or an automatic leak verdict. Use only synthetic benchmark artifacts.
"""
import argparse
import json
from pathlib import Path
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
  counts={'actorShapes':0,'effectShapes':0,'runShapes':0,'attemptRecordShapes':0};names={};run_nodes=set();run_ids=[];off=0
  for i in range(0,len(nodes),ns):
   typ=nt[nodes[i+ti]];name=strings[nodes[i+ni]];count=nodes[i+ei]
   if typ=='object':
    props={strings[edges[j+eni]] for j in range(off,off+count*es,es) if et[edges[j+eti]]=='property'}
    if {'hp','max_hp','kind'}.issubset(props):counts['actorShapes']+=1
    if {'x','y','started','kind'}.issubset(props):counts['effectShapes']+=1
    if {'mission','outcome','at_ms','splits','hp','class'}.issubset(props):counts['attemptRecordShapes']+=1
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
  sample['checkpoints'].append({checkpoint_key:point[checkpoint_key],'elapsedMS':point.get('elapsedMS'),'shapes':counts,'audioAndTimers':names,'runRetainers':retainers,'retainedRunIDs':run_ids,'detached':point['reachable']['detached'],'dom':point['dom'],'usedSize':point['heap']['usedSize']})
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
