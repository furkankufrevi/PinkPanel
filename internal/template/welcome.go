package template

import "fmt"

// DefaultIndexPage returns an HTML welcome page for a newly created domain.
func DefaultIndexPage(domainName string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>%s</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }

  body {
    background: #0a0a0f;
    overflow: hidden;
    height: 100vh;
    font-family: 'Segoe UI', system-ui, -apple-system, sans-serif;
    cursor: crosshair;
  }

  .cube {
    position: absolute;
    width: 50px;
    height: 50px;
    transform-style: preserve-3d;
    animation: float 6s ease-in-out infinite, spin 8s linear infinite;
    pointer-events: none;
  }
  .cube .face {
    position: absolute;
    width: 50px;
    height: 50px;
    border: 1.5px solid;
    background: rgba(232, 67, 147, 0.03);
    backdrop-filter: blur(2px);
  }
  .cube .front  { transform: translateZ(25px); }
  .cube .back   { transform: translateZ(-25px) rotateY(180deg); }
  .cube .left   { transform: translateX(-25px) rotateY(-90deg); }
  .cube .right  { transform: translateX(25px) rotateY(90deg); }
  .cube .top    { transform: translateY(-25px) rotateX(90deg); }
  .cube .bottom { transform: translateY(25px) rotateX(-90deg); }

  @keyframes spin {
    from { transform: rotateX(0) rotateY(0); }
    to   { transform: rotateX(360deg) rotateY(360deg); }
  }
  @keyframes float {
    0%%, 100%% { translate: 0 0; }
    50%%      { translate: 0 -30px; }
  }

  .blob {
    position: fixed;
    width: 400px;
    height: 400px;
    border-radius: 30%% 70%% 70%% 30%% / 30%% 30%% 70%% 70%%;
    animation: morph 8s ease-in-out infinite, blob-move 12s ease-in-out infinite;
    opacity: 0.12;
    filter: blur(60px);
    z-index: 1;
    pointer-events: none;
  }
  .blob-pink {
    background: linear-gradient(135deg, #E84393, #FD79A8);
    top: 15%%; left: 20%%;
  }
  .blob-violet {
    background: linear-gradient(135deg, #6C5CE7, #A29BFE);
    top: 55%%; left: 55%%;
    animation-delay: -4s;
  }
  @keyframes morph {
    0%%   { border-radius: 30%% 70%% 70%% 30%% / 30%% 30%% 70%% 70%%; }
    25%%  { border-radius: 58%% 42%% 28%% 72%% / 55%% 68%% 32%% 45%%; }
    50%%  { border-radius: 50%% 50%% 33%% 67%% / 55%% 27%% 73%% 45%%; }
    75%%  { border-radius: 33%% 67%% 58%% 42%% / 63%% 68%% 32%% 37%%; }
    100%% { border-radius: 30%% 70%% 70%% 30%% / 30%% 30%% 70%% 70%%; }
  }
  @keyframes blob-move {
    0%%, 100%% { transform: translate(0, 0) scale(1); }
    25%%      { transform: translate(80px, -40px) scale(1.1); }
    50%%      { transform: translate(-40px, 80px) scale(0.9); }
    75%%      { transform: translate(40px, 40px) scale(1.05); }
  }

  .trail-dot {
    position: fixed;
    width: 6px;
    height: 6px;
    border-radius: 50%%;
    pointer-events: none;
    z-index: 20;
    animation: trail-fade 0.8s forwards;
    mix-blend-mode: screen;
  }
  @keyframes trail-fade {
    0%%   { transform: scale(1); opacity: 0.8; }
    100%% { transform: scale(2.5); opacity: 0; }
  }

  .scanlines {
    position: fixed;
    inset: 0;
    background: repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0,0,0,0.06) 2px, rgba(0,0,0,0.06) 4px);
    pointer-events: none;
    z-index: 100;
  }

  .matrix-col {
    position: fixed;
    top: -100%%;
    font-size: 13px;
    font-family: 'Courier New', monospace;
    writing-mode: vertical-rl;
    animation: matrix-fall linear infinite;
    opacity: 0.15;
    z-index: 2;
    pointer-events: none;
  }
  @keyframes matrix-fall {
    from { top: -100%%; }
    to   { top: 110%%; }
  }

  .orbit {
    position: fixed;
    top: 50%%;
    left: 50%%;
    width: 220px;
    height: 220px;
    margin: -110px;
    animation: orbit-spin 12s linear infinite;
    z-index: 5;
    pointer-events: none;
  }
  .orbit-dot {
    position: absolute;
    width: 4px;
    height: 4px;
    border-radius: 50%%;
    background: #fff;
    box-shadow: 0 0 8px currentColor, 0 0 16px currentColor;
  }
  @keyframes orbit-spin { to { transform: rotate(360deg); } }

  .center {
    position: fixed;
    top: 50%%;
    left: 50%%;
    transform: translate(-50%%, -50%%);
    z-index: 10;
    text-align: center;
    pointer-events: none;
  }

  .logo {
    width: 90px;
    height: 90px;
    margin: 0 auto 28px;
    border-radius: 22px;
    background: linear-gradient(135deg, #E84393, #6C5CE7);
    display: flex;
    align-items: center;
    justify-content: center;
    animation: logo-float 5s ease-in-out infinite;
    box-shadow: 0 0 40px rgba(232,67,147,0.3), 0 0 80px rgba(232,67,147,0.1);
    position: relative;
    overflow: hidden;
  }
  .logo::before {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: inherit;
    background: linear-gradient(135deg, rgba(255,255,255,0.15), transparent);
  }
  .logo svg {
    width: 48px;
    height: 48px;
    filter: drop-shadow(0 2px 8px rgba(0,0,0,0.3));
  }
  @keyframes logo-float {
    0%%, 100%% { transform: translateY(0); }
    50%% { transform: translateY(-12px); }
  }

  .domain-name {
    font-size: clamp(1.8rem, 5vw, 3rem);
    font-weight: 700;
    color: #fff;
    letter-spacing: -0.02em;
    margin-bottom: 8px;
    position: relative;
    animation: glitch-skew 4s infinite linear alternate-reverse;
  }
  .domain-name::before, .domain-name::after {
    content: attr(data-text);
    position: absolute;
    top: 0; left: 0;
    width: 100%%; height: 100%%;
  }
  .domain-name::before {
    color: #E84393;
    animation: glitch-1 3s infinite linear alternate-reverse;
    clip-path: polygon(0 0, 100%% 0, 100%% 35%%, 0 35%%);
  }
  .domain-name::after {
    color: #6C5CE7;
    animation: glitch-2 2.5s infinite linear alternate-reverse;
    clip-path: polygon(0 65%%, 100%% 65%%, 100%% 100%%, 0 100%%);
  }
  @keyframes glitch-1 {
    0%%   { transform: translate(0); }
    20%%  { transform: translate(-2px, 2px); }
    40%%  { transform: translate(2px, -1px); }
    60%%  { transform: translate(-1px, 1px); }
    80%%  { transform: translate(3px, -2px); }
    100%% { transform: translate(-1px, 1px); }
  }
  @keyframes glitch-2 {
    0%%   { transform: translate(0); }
    20%%  { transform: translate(2px, -2px); }
    40%%  { transform: translate(-2px, 1px); }
    60%%  { transform: translate(1px, -1px); }
    80%%  { transform: translate(-3px, 2px); }
    100%% { transform: translate(1px, -1px); }
  }
  @keyframes glitch-skew {
    0%%   { transform: skew(0deg); }
    20%%  { transform: skew(-0.3deg); }
    40%%  { transform: skew(0.3deg); }
    60%%  { transform: skew(-0.2deg); }
    80%%  { transform: skew(0.2deg); }
    100%% { transform: skew(0deg); }
  }

  .tagline {
    color: rgba(255,255,255,0.35);
    font-size: 0.85rem;
    letter-spacing: 0.3em;
    text-transform: uppercase;
    margin-bottom: 32px;
    animation: flicker 4s infinite;
  }
  @keyframes flicker {
    0%%, 19%%, 21%%, 23%%, 25%%, 54%%, 56%%, 100%% { opacity: 0.35; }
    20%%, 24%%, 55%% { opacity: 0; }
  }

  .badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 8px 20px;
    border-radius: 100px;
    font-size: 0.75rem;
    font-weight: 500;
    background: rgba(232, 67, 147, 0.08);
    color: rgba(253, 121, 168, 0.7);
    border: 1px solid rgba(232, 67, 147, 0.15);
    letter-spacing: 0.05em;
    pointer-events: auto;
    text-decoration: none;
    transition: all 0.3s;
  }
  .badge:hover {
    background: rgba(232, 67, 147, 0.15);
    color: #FD79A8;
    border-color: rgba(232, 67, 147, 0.3);
  }
  .badge svg { width: 14px; height: 14px; }

  .pulse-ring {
    position: absolute;
    border-radius: 50%%;
    border: 1px solid;
    animation: pulse-expand 2s ease-out forwards;
    pointer-events: none;
  }
  @keyframes pulse-expand {
    0%%   { width: 0; height: 0; opacity: 0.8; }
    100%% { width: 200px; height: 200px; opacity: 0; }
  }
</style>
</head>
<body>

<div class="scanlines"></div>
<div class="blob blob-pink"></div>
<div class="blob blob-violet"></div>

<div class="center">
  <div class="logo">
    <svg viewBox="0 0 100 100" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M30 20h28c12 0 22 10 22 22s-10 22-22 22H44v16H30V20z" fill="white"/>
      <circle cx="68" cy="72" r="10" fill="rgba(255,255,255,0.7)"/>
    </svg>
  </div>
  <div class="domain-name" data-text="%s">%s</div>
  <p class="tagline">this site is coming soon</p>
  <a class="badge" href="https://github.com/furkankufrevi/PinkPanel" target="_blank">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 19V5M5 12l7-7 7 7"/></svg>
    Powered by PinkPanel
  </a>
</div>

<div class="orbit">
  <div class="orbit-dot" style="top:0;left:50%%;color:#E84393"></div>
  <div class="orbit-dot" style="top:50%%;right:0;color:#6C5CE7"></div>
  <div class="orbit-dot" style="bottom:0;left:50%%;color:#FD79A8"></div>
  <div class="orbit-dot" style="top:50%%;left:0;color:#A29BFE"></div>
</div>
<div class="orbit" style="width:340px;height:340px;margin:-170px;animation-direction:reverse;animation-duration:18s">
  <div class="orbit-dot" style="top:0;left:50%%;color:#fd79a8"></div>
  <div class="orbit-dot" style="bottom:0;left:50%%;color:#a29bfe"></div>
</div>

<script>
  var colors=['#E84393','#FD79A8','#6C5CE7','#A29BFE','#C22D78','#fff'];
  document.addEventListener('mousemove',function(e){
    var d=document.createElement('div');d.className='trail-dot';
    d.style.left=(e.clientX-3)+'px';d.style.top=(e.clientY-3)+'px';
    d.style.background=colors[Math.floor(Math.random()*colors.length)];
    document.body.appendChild(d);setTimeout(function(){d.remove()},800);
  });
  for(var i=0;i<6;i++){
    var cube=document.createElement('div');cube.className='cube';
    cube.style.left=(Math.random()*90+5)+'%%';
    cube.style.top=(Math.random()*80+10)+'%%';
    cube.style.animationDelay=(-Math.random()*6)+'s';
    cube.style.animationDuration=(6+Math.random()*6)+'s,'+(6+Math.random()*8)+'s';
    var c=colors[Math.floor(Math.random()*4)];
    ['front','back','left','right','top','bottom'].forEach(function(f){
      var face=document.createElement('div');face.className='face '+f;
      face.style.borderColor=c;cube.appendChild(face);
    });
    document.body.appendChild(cube);
  }
  document.addEventListener('click',function(e){
    for(var i=0;i<2;i++){
      var ring=document.createElement('div');ring.className='pulse-ring';
      ring.style.left=e.clientX+'px';ring.style.top=e.clientY+'px';
      ring.style.marginLeft='-100px';ring.style.marginTop='-100px';
      ring.style.borderColor=colors[Math.floor(Math.random()*colors.length)];
      ring.style.animationDelay=(i*0.15)+'s';
      document.body.appendChild(ring);
      setTimeout((function(r){return function(){r.remove()}})(ring),2000);
    }
  });
  var chars='PINKPANEL01';
  for(var i=0;i<12;i++){
    var col=document.createElement('div');col.className='matrix-col';
    col.style.left=(Math.random()*100)+'%%';
    col.style.animationDuration=(6+Math.random()*12)+'s';
    col.style.animationDelay=(-Math.random()*15)+'s';
    col.style.color=colors[Math.floor(Math.random()*4)];
    var t='';for(var j=0;j<25;j++)t+=chars[Math.floor(Math.random()*chars.length)]+'\n';
    col.textContent=t;document.body.appendChild(col);
  }
</script>

</body>
</html>`, domainName, domainName, domainName)
}
