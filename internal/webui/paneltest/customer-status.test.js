import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';

const html = readFileSync(new URL('../assets/customer-status.html', import.meta.url), 'utf8');
const script = html.match(/<script[^>]*>([\s\S]*?)<\/script>/)[1];
const flush = () => new Promise(resolve => setImmediate(resolve));

function element() {
  const classes = new Set();
  return {
    dataset: {}, style: {}, children: [], nodes: [],
    classList: {
      add: name => classes.add(name), remove: name => classes.delete(name),
      contains: name => classes.has(name),
      toggle(name, enabled) { if (enabled) classes.add(name); else classes.delete(name); },
    },
    appendChild(node) { this.nodes.push(node); if (!node.textNode) this.children.push(node); },
    set innerHTML(value) { this.nodes = []; this.children = []; this.text = ''; },
    get textContent() { return this.text || this.nodes.map(node => node.textContent).join(''); },
    set textContent(value) { this.text = String(value); },
  };
}

function customer(lang = 'en') {
  const elements = {}, requests = [], id = 'a'.repeat(48);
  let poll;
  const document = {
    documentElement: {}, body: element(),
    getElementById(name) { return elements[name] ??= element(); },
    querySelectorAll() { return []; }, createElement: element,
    createTextNode(text) { return { textNode: true, textContent: text }; },
  };
  runInNewContext(script, {
    document, location: { pathname: '/status/' + id }, navigator: { language: lang },
    localStorage: { getItem() { return null; }, setItem() {} },
    setInterval(fn) { poll = fn; },
    fetch(url) { return new Promise((resolve, reject) => requests.push({ url, resolve, reject })); },
  });
  return { elements, requests, document, poll: () => poll(), id };
}

const data = used => ({ ports: ['23298', '53835'], state: 'online', mode: 'both', limitBytes: 100, usedBytes: used });
const response = used => ({ ok: true, json: async () => data(used) });

for (const lang of ['fa', 'en']) {
  test(`customer ports remain readable and requests omit the admin path (${lang})`, async () => {
    const page = customer(lang);
    page.requests[0].resolve(response(20));
    await flush();
    assert.equal(page.elements.ports.textContent, '23298, 53835');
    assert.equal(page.requests[0].url, '/api/public/status?id=' + page.id);
    assert.equal(page.document.documentElement.dir, lang === 'fa' ? 'rtl' : 'ltr');
  });
}

test('a delayed older response cannot replace the newest usage', async () => {
  const page = customer();
  page.poll();
  page.requests[1].resolve(response(80));
  await flush();
  page.requests[0].resolve(response(20));
  await flush();
  assert.equal(page.elements.used.textContent, '80 B');
  assert.equal(page.elements.remaining.textContent, '20 B');
});

test('a delayed older error cannot hide a newer successful status', async () => {
  const page = customer();
  page.elements.lang.onclick();
  page.requests[1].resolve(response(80));
  await flush();
  page.requests[0].reject(new Error('old request failed'));
  await flush();
  assert.equal(page.elements.content.hidden, false);
  assert.equal(page.elements.error.classList.contains('show'), false);
});

test('a delayed JSON body cannot replace a newer response', async () => {
  const page = customer();
  let finishBody;
  page.requests[0].resolve({ ok: true, json: () => new Promise(resolve => { finishBody = resolve; }) });
  await flush();
  page.poll();
  page.requests[1].resolve(response(80));
  await flush();
  finishBody(data(20));
  await flush();
  assert.equal(page.elements.used.textContent, '80 B');
});
