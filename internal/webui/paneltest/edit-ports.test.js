import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';

const source = readFileSync(new URL('../panel/js/views/edit.js', import.meta.url), 'utf8')
  .replace(/^import .*;$/gm, '').replace(/^export /gm, '');
const { editPayload, keepDirectTab } = runInNewContext(source + '\n({editPayload, keepDirectTab});');
function root(ports) {
  const fields = { trafficLimitGB: '4000', trafficLimitMode: 'download', ports };
  return { querySelector(selector) {
    const name = selector.match(/name="([^"]+)"/)[1];
    return name in fields ? { value: fields[name] } : null;
  } };
}
for (const ports of ['23298, 53835, 2082', '53835']) {
  test(`direct edit submits the changed nonempty ports: ${ports}`, () => {
    const payload = editPayload('iran-xdi', root(ports), { holdsPorts: true, ports: '23298,53835' }, true);
    assert.equal(payload.direct.ports, ports);
    assert.equal(payload.direct.trafficLimitGB, 4000);
    assert.equal(payload.direct.trafficLimitMode, 'download');
  });
}
test('kharej edits never send an Iran port list', () => {
  const payload = editPayload('kharej-xdi', root('23298'), { holdsPorts: false }, true);
  assert.equal(Object.hasOwn(payload.direct, 'ports'), false);
});

test('the Ports tab stays accessible when its label contains a count', () => {
  const html = readFileSync(new URL('../panel/views/edit.html', import.meta.url), 'utf8');
  const button = html.match(/<button([^>]*data-edit-tab="ports"[^>]*)>([\s\S]*?)<\/button>/);
  assert.ok(button, 'the real template must identify its Ports tab');
  assert.match(button[2], /<span class="n">/);
  const tab = { dataset: { editTab: 'ports' }, textContent: 'Ports4', classList: { contains: () => false } };
  assert.equal(keepDirectTab(tab, true), true);
  assert.equal(keepDirectTab(tab, false), false);
});
