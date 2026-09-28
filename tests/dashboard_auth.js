const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const html = fs.readFileSync('dashboard.html', 'utf8');
const start = html.indexOf('function decodeStoredAuth(');
const end = html.indexOf('function showLogin(', start);
assert(start >= 0 && end > start);
const source = html.slice(start, end);
const host = '10.10.10.121:18317';
const agent = 'test-browser';
const key = 'synthetic-test-key';
function encode(version) {
  const mask = new TextEncoder().encode('cli-proxy-api-webui::secure-storage|' + (version === 2 ? 'v2|' + host : host + '|' + agent));
  const data = new TextEncoder().encode(JSON.stringify({state: {managementKey: key, rememberPassword: true, apiBase: 'http://10.10.10.121:8317'}}));
  return `enc::v${version}::` + btoa(String.fromCharCode(...data.map((byte, i) => byte ^ mask[i % mask.length])));
}
for (const version of [1, 2]) {
  const storage = {'cli-proxy-auth': encode(version)};
  const session = {'headroom-management-key': 'old-tab-key'};
  const context = {
    TextEncoder, TextDecoder, atob,
    location: {host}, navigator: {userAgent: agent},
    localStorage: {getItem: name => storage[name] || null},
    sessionStorage: {getItem: name => session[name] || null},
  };
  vm.runInNewContext(source, context);
  assert.equal(vm.runInNewContext('readManagementKey()', context), key);
  session['headroom-management-key-override'] = '1';
  assert.equal(vm.runInNewContext('readManagementKey()', context), 'old-tab-key');
}
console.log('Saved CPA login read from v1 and v2 formats and preferred over a stale tab key');
