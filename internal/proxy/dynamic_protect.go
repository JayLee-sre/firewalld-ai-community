package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// injectDynamicScript injects a JavaScript snippet into HTML that randomizes
// element attributes on each page load, defeating static analysis and scraping.
func injectDynamicScript(html []byte) []byte {
	htmlStr := string(html)

	// Generate a unique random seed for this request
	seed := randHex(8)

	script := fmt.Sprintf(`<script>(function(){
var s="%s",c=0;
function h(x){for(var i=0,r=0;i<x.length;i++)r=((r<<5)-r+x.charCodeAt(i))|0;return Math.abs(r).toString(36)}
function r(n){var a="",i;for(i=0;i<n;i++)a+=String.fromCharCode(97+Math.floor(Math.random()*26));return a}
function m(el){
  var id=el.getAttribute("id");
  if(id&&!el.getAttribute("data-o")){el.setAttribute("data-o",id);el.setAttribute("id",id+"_"+h(s+(c++)))}
  var cls=el.getAttribute("class");
  if(cls&&!el.getAttribute("data-oc")){
    el.setAttribute("data-oc",cls);
    var parts=cls.split(/\s+/).map(function(p){return p+"_"+r(4)});
    el.setAttribute("class",parts.join(" "))
  }
  for(var j=el.attributes.length-1;j>=0;j--){
    var a=el.attributes[j];
    if(a.name.indexOf("data-")===0&&a.name!=="data-o"&&a.name!=="data-oc"&&a.name.indexOf("data-r")!==0)continue;
    if(a.name==="data-r")continue;
  }
}
var d=100+Math.floor(Math.random()*200);
setTimeout(function(){
  document.querySelectorAll("[id],[class]").forEach(m);
  var ps=document.querySelectorAll("p,span,div,li");
  ps.forEach(function(el,i){
    if(Math.random()>.7)el.setAttribute("data-r",r(6)+"_"+i);
  });
},d);
})();</script>`, seed)

	// Inject before </body>, or before </html>, or append
	if idx := strings.LastIndex(htmlStr, "</body>"); idx >= 0 {
		return []byte(htmlStr[:idx] + script + htmlStr[idx:])
	}
	if idx := strings.LastIndex(htmlStr, "</html>"); idx >= 0 {
		return []byte(htmlStr[:idx] + script + htmlStr[idx:])
	}
	return append(html, []byte(script)...)
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
