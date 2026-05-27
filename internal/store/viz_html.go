package store

const visualizerHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Glyph Mesh Visualizer</title>
<style>
body{margin:0;font:14px system-ui,-apple-system,Segoe UI,sans-serif;background:#101216;color:#eceff4}
.app{display:grid;grid-template-columns:260px 1fr 360px;height:100vh}
aside,.inspect{padding:16px;background:#171a21;overflow:auto}
main{position:relative;overflow:hidden}
h1,h2{font-size:16px;margin:0 0 12px}
input{box-sizing:border-box;width:100%;padding:8px;background:#0f1117;border:1px solid #343946;color:#fff;border-radius:6px}
label{display:block;margin:8px 0;color:#c9d1dc}
.summary{display:grid;grid-template-columns:1fr 1fr;gap:6px;margin:12px 0}
.pill{padding:6px 8px;background:#252a34;border-radius:6px}
svg{width:100%;height:100%;background:#0f1117}
.node{cursor:pointer}
.edge{stroke:#3d4657;stroke-width:1.2;opacity:.75}
.label{fill:#dfe6f3;font-size:11px;pointer-events:none}
pre{white-space:pre-wrap;background:#0f1117;padding:10px;border-radius:6px;border:1px solid #303642}
button{width:100%;padding:8px;margin:4px 0;background:#2f6fed;color:white;border:0;border-radius:6px}
.empty{position:absolute;inset:0;display:grid;place-items:center;color:#8e98a8}
</style>
</head>
<body>
<div class="app">
<aside>
<h1>Glyph Mesh</h1>
<input id="search" placeholder="Search mesh">
<div class="summary" id="summary"></div>
<h2>Types</h2>
<div id="filters"></div>
</aside>
<main><svg id="graph"></svg><div class="empty" id="empty">Loading graph...</div></main>
<section class="inspect">
<h2>Inspector</h2>
<pre id="details">Select a node.</pre>
</section>
</div>
<script>
window.__GLYPH_GRAPH__ = __GLYPH_GRAPH_JSON__;
const colors={store:"#f2cc60",realm:"#7cc7ff",work:"#a4f07a",snapshot:"#b39cff",publication:"#ffb86b",source:"#69dbb8",content:"#8aa1ff",claim:"#ff7eb6",conflict:"#ff5c5c",hook_run:"#d6de6a",remote:"#60d4f2",mount:"#c792ea"};
let graph, activeTypes=new Set(), selected=null;
loadGraph();
function loadGraph(){
  if(window.__GLYPH_GRAPH__){init(window.__GLYPH_GRAPH__);return;}
  fetch("graph.json").then(r=>r.json()).then(init).catch(err=>{document.getElementById("empty").textContent="Could not load graph.json: "+err.message;});
}
function init(g){graph=g;activeTypes=new Set(g.nodes.map(n=>n.type));setup();draw();}
function setup(){
  const summary=document.getElementById("summary");
  Object.entries(graph.summary).sort().forEach(([k,v])=>{const d=document.createElement("div");d.className="pill";d.textContent=k+": "+v;summary.appendChild(d);});
  const filters=document.getElementById("filters");
  [...activeTypes].sort().forEach(t=>{const l=document.createElement("label");const c=document.createElement("input");c.type="checkbox";c.checked=true;c.style.width="auto";c.onchange=()=>{c.checked?activeTypes.add(t):activeTypes.delete(t);draw();};l.append(c," "+t);filters.appendChild(l);});
  document.getElementById("search").oninput=draw;
}
function draw(){
  const q=document.getElementById("search").value.toLowerCase();
  const nodes=graph.nodes.filter(n=>activeTypes.has(n.type)&&match(n,q));
  document.getElementById("empty").style.display=nodes.length?"none":"grid";
  if(!nodes.length){document.getElementById("empty").textContent="No matching nodes.";return;}
  const ids=new Set(nodes.map(n=>n.id));
  const edges=graph.edges.filter(e=>ids.has(e.from)&&ids.has(e.to));
  const svg=document.getElementById("graph");svg.innerHTML="";
  const w=svg.clientWidth||800,h=svg.clientHeight||600,cx=w/2,cy=h/2;
  const pos=new Map(nodes.map((n,i)=>{const a=i/nodes.length*Math.PI*2;const r=Math.min(w,h)*(.18+.32*((i%5)/5));return[n.id,{x:cx+Math.cos(a)*r,y:cy+Math.sin(a)*r}]}));
  edges.forEach(e=>{const a=pos.get(e.from),b=pos.get(e.to);if(!a||!b)return;line(svg,a.x,a.y,b.x,b.y);});
  nodes.forEach(n=>{const p=pos.get(n.id);circle(svg,p.x,p.y,8,colors[n.type]||"#ccc",()=>inspect(n));text(svg,p.x+11,p.y+4,n.label.slice(0,34));});
}
function match(n,q){return !q||JSON.stringify(n).toLowerCase().includes(q);}
function line(svg,x1,y1,x2,y2){const e=document.createElementNS("http://www.w3.org/2000/svg","line");e.setAttribute("class","edge");e.setAttribute("x1",x1);e.setAttribute("y1",y1);e.setAttribute("x2",x2);e.setAttribute("y2",y2);svg.appendChild(e);}
function circle(svg,x,y,r,fill,click){const e=document.createElementNS("http://www.w3.org/2000/svg","circle");e.setAttribute("class","node");e.setAttribute("cx",x);e.setAttribute("cy",y);e.setAttribute("r",r);e.setAttribute("fill",fill);e.onclick=click;svg.appendChild(e);}
function text(svg,x,y,s){const e=document.createElementNS("http://www.w3.org/2000/svg","text");e.setAttribute("class","label");e.setAttribute("x",x);e.setAttribute("y",y);e.textContent=s;svg.appendChild(e);}
function inspect(n){document.getElementById("details").textContent=JSON.stringify(n,null,2);}
</script>
</body>
</html>
`
