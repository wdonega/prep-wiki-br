// Comportamento das páginas do ZIM de canal (montar.py). Sem dependências: roda no leitor do Kiwix.
document.addEventListener('click',function(e){
  var b=e.target.closest('button[data-t]');if(!b)return;
  var v=document.querySelector('video');v.currentTime=+b.dataset.t;v.play();window.scrollTo(0,0);
});
var t=new URLSearchParams(location.search).get('t');
if(t){var v=document.querySelector('video');v&&v.addEventListener('loadedmetadata',function(){v.currentTime=+t},{once:true});}
// descrição da playlist: toque abre e fecha o texto inteiro
var d=document.querySelector('.capa .desc');d&&d.addEventListener('click',function(){d.classList.toggle('aberta')});
// "Reproduzir tudo" (?tudo=1): começa sozinho e, no fim, vai para o próximo vídeo
if(new URLSearchParams(location.search).has('tudo')){
  var v=document.querySelector('video'),p=document.getElementById('proximo');
  if(v){v.play().catch(function(){});
    v.addEventListener('ended',function(){if(p)location.href=p.getAttribute('href')+'?tudo=1'});}
  if(p)p.href+='?tudo=1';
}
