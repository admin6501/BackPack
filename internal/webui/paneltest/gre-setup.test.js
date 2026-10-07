import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';
const source = readFileSync(new URL('../panel/js/views/add.js', import.meta.url), 'utf8')
 .replace(/^import .*;$/gm,'').replace(/^export /gm,'');
const {usesL3, setupCarrier, greSetupMode} = runInNewContext(source+'\n({usesL3,setupCarrier,greSetupMode});');
for(const direction of ['direct','reverse']) for(const side of ['server','client']) {
 test(`GRE ${direction} on ${side} preserves L3 API and initiation`,()=>{
 const chosen={side,direction,transport:direction==='reverse'?'gre-fou':'tcp',carrier:direction==='direct'?'gre-fou':'pck'};
 assert.equal(usesL3(chosen),true);
 assert.equal(setupCarrier(chosen),'gre-fou');
 assert.equal(greSetupMode(chosen), (side==='server')!==(direction==='reverse')?'dial':'listen');
 });
}
test('ordinary reverse transports retain the stream API',()=>{
 for(const transport of ['tcp','udp','ws']) assert.equal(usesL3({direction:'reverse',transport,carrier:'gre-fou'}),false);
});
test('GRE follows the selected direction without a second selector',()=>{
 assert.doesNotMatch(source,/fouDirection|fouModeGroup/);
 assert.match(source,/payload\.carrier = setupCarrier\(chosen\)/);
 assert.match(source,/payload\.mode = greSetupMode\(chosen\)/);
});

const {showIPv6Family,l3Listens} = runInNewContext(source+'\n({showIPv6Family,l3Listens});');
test('IPv6 family selection follows the actual L3 listener',()=>{
 for(const carrier of ['xdi','udp','quic','gre-fou','pck','sni','spoof']) {
  for(const side of ['server','client']) {
   const c={direction:'direct',side,carrier};
   assert.equal(l3Listens(c),side==='client');
   assert.equal(showIPv6Family(c),side==='client'&&['xdi','udp','quic','gre-fou'].includes(carrier));
  }
 }
 for(const side of ['server','client']) {
  const c={direction:'reverse',side,transport:'gre-fou'};
  assert.equal(showIPv6Family(c),side==='server');
 }
 assert.equal(showIPv6Family({direction:'reverse',side:'server',transport:'tcp'}),false);
});
