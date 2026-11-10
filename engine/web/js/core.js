/* SermonROAD — 공통: 엔진과 이야기하기 · 상태 · 작은 도구들 · 알림 · 창(모달) · 길 찾기
   made by. Nevertheless_D */
'use strict';

const TOKEN = new URLSearchParams(location.search).get('t') || '';
const S = {
  settings: {}, books: [], data: { sermons: [], series: [], illus: [], rules: [], trash: [] },
  rev: 0, lib: null, libRev: 0, update: null, route: { name: 'map', q: {} }, screen: null,
};
const Screens = {};

// ---------------------------------------------------------------- 엔진
async function api(path, body, opts = {}) {
  const init = { headers: { 'x-token': TOKEN } };
  if (body instanceof FormData) {
    init.method = 'POST'; init.body = body;
  } else if (body !== undefined) {
    init.method = 'POST'; init.body = JSON.stringify(body); init.headers['content-type'] = 'application/json';
  }
  let res;
  try {
    res = await fetch(path, init);
  } catch (e) {
    throw new Error('앱 엔진에 연결하지 못했어요. 잠시 뒤 다시 시도해 주세요');
  }
  let out = null;
  try { out = await res.json(); } catch (e) { /* 빈 답 */ }
  if (!res.ok) throw new Error((out && out.error) || ('문제가 생겼어요 (' + res.status + ')'));
  return out;
}

async function loadData() {
  const d = await api('/api/data');
  S.data = d; S.rev = d.rev;
  indexData();
}

// 빠르게 찾을 수 있게 설교마다 날짜 객체 · 책 번호를 붙여 둬요
function indexData() {
  const D = S.data;
  D.byId = {};
  for (const s of D.sermons) {
    D.byId[s.id] = s;
    s._d = parseISO(s.date);
    s._ref = refText(s);
  }
  D.sermons.sort((a, b) => (a.date + (a.time || '')).localeCompare(b.date + (b.time || '')));
  D.seriesById = {}; for (const x of D.series) D.seriesById[x.id] = x;
  D.illusById = {}; for (const x of D.illus) D.illusById[x.id] = x;
}

// ---------------------------------------------------------------- 작은 도구
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const $ = (sel, el = document) => el.querySelector(sel);
const $$ = (sel, el = document) => Array.from(el.querySelectorAll(sel));
const pad = (n) => String(n).padStart(2, '0');
const DOW = ['일', '월', '화', '수', '목', '금', '토'];
function parseISO(s) { const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(s || ''); return m ? new Date(+m[1], +m[2] - 1, +m[3]) : null; }
function iso(d) { return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()); }
function ymd(y, m, d) { return y + '-' + pad(m) + '-' + pad(d); }
function today() { const d = new Date(); d.setHours(0, 0, 0, 0); return d; }
function addDays(d, n) { const x = new Date(d); x.setDate(x.getDate() + n); return x; }
function fmtDate(s, withDow = false) { const d = parseISO(s); if (!d) return s || ''; return d.getFullYear() + '.' + pad(d.getMonth() + 1) + '.' + pad(d.getDate()) + (withDow ? ' (' + DOW[d.getDay()] + ')' : ''); }
function fmtShort(s) { const d = parseISO(s); return d ? (d.getMonth() + 1) + '.' + d.getDate() + ' ' + DOW[d.getDay()] : ''; }
function dayName(d) { return d.getDay() === 0 ? '주일' : DOW[d.getDay()] + '요일'; }
function debounce(fn, ms) { let t; return (...a) => { clearTimeout(t); t = setTimeout(() => fn(...a), ms); }; }
function uniq(a) { return Array.from(new Set(a)); }
function plural(n, unit) { return n + unit; }
function hexA(hex, a) { return hex + Math.round(a * 255).toString(16).padStart(2, '0'); }
function isDark() { return document.documentElement.dataset.theme === 'dark'; }
function svgIcon(path, size = 16) { return `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${path}</svg>`; }
const IC = {
  plus: '<path d="M12 5v14M5 12h14"/>', left: '<path d="m15 6-6 6 6 6"/>', right: '<path d="m9 6 6 6-6 6"/>',
  ext: '<path d="M7 17 17 7M8 7h9v9"/>', search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
  cal: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/>',
  bulb: '<path d="M9 18h6M10 21h4M12 3a6 6 0 0 0-4 10.5c.8.8 1 1.6 1 2.5h6c0-.9.2-1.7 1-2.5A6 6 0 0 0 12 3Z"/>',
  warn: '<path d="M12 9v4M12 17h.01M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z"/>',
  trash: '<path d="M3 6h18M8 6V4h8v2M6 6l1 14h10l1-14"/>', x: '<path d="M18 6 6 18M6 6l12 12"/>',
  copy: '<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15V5a2 2 0 0 1 2-2h10"/>',
  file: '<path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9Z"/><path d="M14 3v6h6"/>',
  folder: '<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/>',
  book: '<path d="M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2V5Z"/><path d="M4 19a2 2 0 0 1 2-2h13v4H6a2 2 0 0 1-2-2Z"/>',
};

// ---------------------------------------------------------------- 성경
function bookOf(n) { return S.books[n - 1]; }
function bibleURL(p) {
  const b = bookOf(p && p.book);
  return b ? `https://www.biblenote.kr/read.html?book=${b.code}&chapter=${p.c1 || 1}` : 'https://www.biblenote.kr/';
}
function refText(s) {
  const ok = (s.passages || []).filter((p) => p.book > 0).map((p) => p.ref);
  return ok.length ? ok.join('; ') : (s.passageText || '');
}
function lo(p) { return p.v1 === 0 ? p.c1 * 1000 + 1 : p.c1 * 1000 + p.v1; }
function hi(p) {
  const c2 = p.c2 || p.c1;
  if (p.v2 === 0) return (p.v1 !== 0 && c2 === p.c1) ? p.c1 * 1000 + p.v1 : c2 * 1000 + 999;
  return c2 * 1000 + p.v2;
}
function overlap(a, b) { return a.book > 0 && a.book === b.book && lo(a) <= hi(b) && lo(b) <= hi(a); }
function sermonsOverlapping(passages, exceptId) {
  const out = [];
  for (const s of S.data.sermons) {
    if (s.id === exceptId) continue;
    if ((s.passages || []).some((p) => passages.some((q) => overlap(p, q)))) out.push(s);
  }
  return out;
}

// ---------------------------------------------------------------- 종류 · 상태 · 절기
const KINDS = ['주일', '주일 오후', '새벽', '수요', '금요', '성찬', '부서', '수련회', '그 외'];
const KIND_DETAIL = { '부서': '어느 부서 (예: 청년부, 중고등부, 주일학교)', '수련회': '어떤 수련회 (예: 여름 수련회, 청년 수련회)', '그 외': '예: 장례 예배, 결혼 예배, 임직식, 부흥회, 심방' };
const KC = { '주일': '#4C7FF0', '주일 오후': '#7A6CF2', '새벽': '#D9A21E', '수요': '#2FA463', '금요': '#E8742F', '성찬': '#D9434A', '부서': '#1DA6B5', '수련회': '#B5832D', '그 외': '#8A90A6' };
const STATUSES = ['본문 미정', '준비 중', '원고 완료', '설교함'];
function kindLabel(s) {
  const o = (s.kindOther || '').trim();
  if (!o) return s.kind;
  if (s.kind === '그 외') return o;
  if (s.kind === '부서' || s.kind === '수련회') return s.kind + ' · ' + o;
  return s.kind;
}
function kindChip(k, label) { return `<span class="kind" style="background:${hexA(KC[k] || '#8A90A6', isDark() ? 0.22 : 0.12)};color:${KC[k] || '#8A90A6'}">${esc(label || k)}</span>`; }
function statusDot(st) {
  const m = { '본문 미정': 'border:1.5px dashed var(--faint)', '준비 중': 'background:var(--prep)', '원고 완료': 'background:var(--ready)', '설교함': 'background:var(--done)' };
  return `<i style="${m[st] || m['준비 중']}"></i>`;
}
function statusHtml(st) { return `<span class="st">${statusDot(st)}${esc(st)}</span>`; }

// 교회력 · 절기 (날짜 → 이름). 종교개혁주일 = 10월 마지막 주일, 추수감사주일 = 11월 셋째 주일, 맥추감사주일 = 7월 첫 주일
const seasonCache = {};
function easter(y) {
  const a = y % 19, b = Math.floor(y / 100), c = y % 100, d = Math.floor(b / 4), e = b % 4, f = Math.floor((b + 8) / 25),
    g = Math.floor((b - f + 1) / 3), h = (19 * a + b - d - g + 15) % 30, i = Math.floor(c / 4), k = c % 4,
    l = (32 + 2 * e + 2 * i - h - k) % 7, m = Math.floor((a + 11 * h + 22 * l) / 451),
    month = Math.floor((h + l - 7 * m + 114) / 31), day = ((h + l - 7 * m + 114) % 31) + 1;
  return new Date(y, month - 1, day);
}
function nthSunday(y, m, n) { const d = new Date(y, m - 1, 1); while (d.getDay() !== 0) d.setDate(d.getDate() + 1); d.setDate(d.getDate() + 7 * (n - 1)); return d; }
function lastSunday(y, m) { const d = new Date(y, m, 0); while (d.getDay() !== 0) d.setDate(d.getDate() - 1); return d; }
function seasonsOfYear(y) {
  if (seasonCache[y]) return seasonCache[y];
  const map = {};
  const put = (d, name) => { const k = iso(d); if (!map[k]) map[k] = name; };
  const E = easter(y);
  put(new Date(y, 0, 1), '신년'); put(new Date(y, 0, 6), '주현절');
  put(addDays(E, -46), '사순절 시작');
  for (let i = 1; i <= 5; i++) put(addDays(E, -49 + 7 * i), '사순절 ' + i + '주');
  put(addDays(E, -7), '종려주일');
  for (let i = -6; i <= -1; i++) put(addDays(E, i), i === -2 ? '성금요일' : '고난주간');
  put(E, '부활절'); put(addDays(E, 39), '승천일'); put(addDays(E, 49), '성령강림절'); put(addDays(E, 56), '삼위일체주일');
  put(nthSunday(y, 7, 1), '맥추감사주일');
  put(nthSunday(y, 10, 1), '세계성찬주일');
  put(lastSunday(y, 10), '종교개혁주일');
  put(nthSunday(y, 11, 3), '추수감사주일');
  const xmas = new Date(y, 11, 25);
  const adv4 = addDays(xmas, -(xmas.getDay() === 0 ? 7 : xmas.getDay()));
  ['대림절 첫째 주일', '대림절 둘째 주일', '대림절 셋째 주일', '대림절 넷째 주일'].forEach((n, i) => put(addDays(adv4, -7 * (3 - i)), n));
  put(xmas, '성탄절'); put(new Date(y, 11, 31), '송구영신');
  seasonCache[y] = map;
  return map;
}
function seasonOf(d) { if (!d) return ''; return seasonsOfYear(d.getFullYear())[iso(d)] || ''; }
// 절기 이름 → 설교 기록의 절기 고르기 이름
const SEASON_OPTIONS = ['없음', '대림절', '성탄절', '주현절', '사순절', '고난주간', '부활절', '승천일', '성령강림절', '맥추감사절', '종교개혁주일', '추수감사절', '송구영신', '직접 입력'];
function seasonOption(label) {
  if (!label) return '';
  if (label.startsWith('대림절')) return '대림절';
  if (label.startsWith('사순절') || label === '종려주일') return label === '종려주일' ? '고난주간' : '사순절';
  if (label === '성금요일' || label === '고난주간') return '고난주간';
  if (label === '맥추감사주일') return '맥추감사절';
  if (label === '추수감사주일') return '추수감사절';
  if (SEASON_OPTIONS.includes(label)) return label;
  return '';
}

// ---------------------------------------------------------------- 알림 · 창
function toast(msg, opts = {}) {
  const el = document.createElement('div');
  el.className = 'toast' + (opts.err ? ' err' : '');
  el.innerHTML = `<span>${esc(msg)}</span>` + (opts.action ? `<button type="button">${esc(opts.action)}</button>` : '');
  if (opts.action) el.querySelector('button').onclick = () => { el.remove(); opts.onAction && opts.onAction(); };
  $('#toasts').appendChild(el);
  setTimeout(() => el.remove(), opts.ms || (opts.action ? 7000 : 3200));
}
function fail(e) { console.error(e); toast(e.message || String(e), { err: true, ms: 5200 }); }

// modal({title, body(html), wide, buttons:[{label, kind, value}], onOpen(el)}) → Promise(value)
function modal(o) {
  return new Promise((resolve) => {
    const root = $('#modal-root');
    const back = document.createElement('div');
    back.className = 'back';
    const btns = (o.buttons || [{ label: '확인', kind: 'brand', value: true }]).map((b, i) =>
      `<button type="button" class="btn ${b.kind || ''}" data-i="${i}">${esc(b.label)}</button>`).join('');
    back.innerHTML = `<div class="modal ${o.wide ? 'wide' : ''}" role="dialog" aria-modal="true" aria-label="${esc(o.title || '')}">
      ${o.title ? `<h2>${esc(o.title)}</h2>` : ''}<div class="mbody col">${o.body || ''}</div><div class="foot">${btns}</div></div>`;
    root.appendChild(back);
    const el = back.querySelector('.modal');
    const close = (v) => { back.remove(); document.removeEventListener('keydown', onKey, true); resolve(v); };
    const onKey = (e) => {
      if (e.key === 'Escape') { e.stopPropagation(); close(undefined); }
      if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) { const b = el.querySelector('.foot .btn.brand'); if (b) { e.preventDefault(); b.click(); } }
    };
    document.addEventListener('keydown', onKey, true);
    back.addEventListener('mousedown', (e) => { if (e.target === back) close(undefined); });
    $$('.foot .btn', el).forEach((b) => b.onclick = async () => {
      const def = o.buttons ? o.buttons[+b.dataset.i] : { value: true };
      if (def.check) { const ok = await def.check(el); if (!ok) return; }
      close(typeof def.value === 'function' ? def.value(el) : def.value);
    });
    el.close = close;
    if (o.onOpen) o.onOpen(el, close);
    const first = el.querySelector('input, textarea, select');
    if (first && o.focus !== false) setTimeout(() => first.focus(), 30);
  });
}
function confirmBox(title, text, okLabel = '확인', danger = false) {
  return modal({ title, body: `<p style="margin:0;line-height:1.65;color:var(--muted)">${text}</p>`,
    buttons: [{ label: '취소', value: false }, { label: okLabel, kind: danger ? 'danger' : 'brand', value: true }] });
}

async function copyText(text) {
  try {
    if (window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.sermonroad) {
      await window.webkit.messageHandlers.sermonroad.postMessage({ type: 'copy', text });
      return true;
    }
  } catch (e) { /* 아래 방법으로 */ }
  try { await navigator.clipboard.writeText(text); return true; } catch (e) { /* 아래 방법으로 */ }
  const ta = document.createElement('textarea'); ta.value = text; document.body.appendChild(ta); ta.select();
  const ok = document.execCommand('copy'); ta.remove(); return ok;
}
async function chooseFolder(prompt, start) {
  if (window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.sermonroad) {
    const r = await window.webkit.messageHandlers.sermonroad.postMessage({ type: 'chooseFolder', prompt, start: start || '' });
    return r && r.path ? r.path : '';
  }
  // 브라우저에서 시험할 때
  const r = await modal({ title: prompt, body: `<label class="field">폴더 위치<input class="input" id="fp" value="${esc(start || '')}"></label>`,
    buttons: [{ label: '취소', value: '' }, { label: '고르기', kind: 'brand', value: (el) => $('#fp', el).value.trim() }] });
  return r || '';
}
function openExternal(url) { window.open(url, '_blank'); }

// ---------------------------------------------------------------- 길 찾기 (#map, #sermon?id=…)
const NAV = [
  { name: 'map', label: '설교 지도', icon: '<rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/>' },
  { name: 'calendar', label: '설교 달력', icon: '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/>' },
  { name: 'records', label: '설교 기록', icon: '<path d="M8 6h12M8 12h12M8 18h12M4 6h.01M4 12h.01M4 18h.01"/>', also: ['sermon', 'import'] },
  { name: 'series', label: '강해 시리즈', icon: '<path d="M12 3 3 8l9 5 9-5-9-5ZM3 13l9 5 9-5"/>' },
  { name: 'confess', label: '요리문답 · 신앙고백', icon: '<path d="M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2V5Z"/><path d="M4 19a2 2 0 0 1 2-2h13v4H6a2 2 0 0 1-2-2ZM11.5 7v6M9 9.4h5"/>' },
  { name: 'illus', label: '예화 보관함', icon: '<path d="M7 8h10M7 12h6M5 4h14a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-7l-5 4v-4H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2Z"/>' },
  { name: 'helper', label: '설교 도우미', icon: IC.bulb },
  { name: 'stats', label: '통계 · 백업', icon: '<path d="M4 20V10M10 20V4M16 20v-7M22 20H2"/>' },
];
function parseHash() {
  const h = location.hash.replace(/^#/, '');
  const [name, qs] = h.split('?');
  const q = {};
  new URLSearchParams(qs || '').forEach((v, k) => { q[k] = v; });
  return { name: name || 'map', q };
}
function go(name, q) {
  const qs = q ? '?' + new URLSearchParams(Object.entries(q).filter(([, v]) => v !== undefined && v !== null && v !== '')).toString() : '';
  const h = '#' + name + (qs === '?' ? '' : qs);
  if (location.hash === h) render(); else location.hash = h;
}
function renderNav() {
  const cur = S.route.name;
  $('#nav').innerHTML = NAV.map((n) => {
    const on = cur === n.name || (n.also || []).includes(cur);
    return `<a class="nav-item ${on ? 'on' : ''}" href="#${n.name}" data-nav="${n.name}"><svg viewBox="0 0 24 24" aria-hidden="true">${n.icon}</svg>${esc(n.label)}</a>`;
  }).join('');
  $$('#side > .nav-item').forEach((a) => a.classList.toggle('on', a.dataset.nav === cur));
}
let lastRouteKey = '';
async function render(force) {
  const r = parseHash();
  const key = location.hash;
  const scr = Screens[r.name] || Screens.map;
  if (S.screen && S.screen.canLeave && key !== lastRouteKey && !force) {
    const ok = await S.screen.canLeave();
    if (!ok) { history.replaceState(null, '', lastRouteKey || '#map'); return; }
  }
  const changed = key !== lastRouteKey;
  lastRouteKey = key;
  S.route = r;
  if (S.screen && S.screen !== scr && S.screen.leave) S.screen.leave();
  S.screen = scr;
  renderNav();
  const main = $('#main');
  main.innerHTML = '';
  const page = document.createElement('div');
  page.className = 'page';
  main.appendChild(page);
  try {
    await scr.render(page, r.q, changed);
  } catch (e) { fail(e); }
  if (changed) main.scrollTop = 0;
}
// 데이터가 바뀌었을 때: 지금 화면을 다시 그려요 (입력 중인 화면은 화면이 알아서)
function refresh() {
  if (S.screen && S.screen.refresh) { S.screen.refresh(); return; }
  const main = $('#main'); const top = main.scrollTop;
  render(true).then(() => { main.scrollTop = top; });
}
function footer() { return '<footer class="made">made by. Nevertheless_D</footer>'; }

// ---------------------------------------------------------------- 화면 모드
function applyTheme() {
  const a = S.settings.appearance || 'system';
  const dark = a === 'dark' || (a === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.dataset.theme = dark ? 'dark' : 'light';
}
matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => { applyTheme(); refresh(); });

// 화면 오류는 진단 기록으로
const clientErrors = [];
window.addEventListener('error', (e) => { clientErrors.push(String(e.message) + ' @' + (e.filename || '').split('/').pop() + ':' + e.lineno); });
window.addEventListener('unhandledrejection', (e) => { clientErrors.push('promise: ' + (e.reason && e.reason.message || e.reason)); });
setInterval(() => { if (clientErrors.length) api('/api/clientlog', { items: clientErrors.splice(0) }).catch(() => {}); }, 5000);

/* 설교 도우미 AI (Claude · ChatGPT · 제미나이) — 자세한 상태는 S.claude (/api/claude) */
const AI_LIST = [['claude', 'Claude', 'Anthropic'], ['chatgpt', 'ChatGPT', 'OpenAI'], ['gemini', '제미나이', 'Google']];
const AI_CHAT = { claude: 'https://claude.ai/new', chatgpt: 'https://chatgpt.com/', gemini: 'https://gemini.google.com/app' };
const AI_COLOR = { claude: '#D97757', chatgpt: '#10A37F', gemini: '#4F7FE8' };
function aiGap(n) { return /[가-힣]$/.test(n || '') ? '' : ' '; } // ‘Claude 가’ · ‘제미나이가’
function curAI() { return (S.settings && S.settings.ai) || 'claude'; }
function aiName(id) { const x = AI_LIST.find((a) => a[0] === (id || curAI())); return x ? x[1] : 'Claude'; }
function aiInfo(id) {
  id = id || curAI();
  return ((S.claude && S.claude.providers) || []).find((p) => p.id === id) || { id, name: aiName(id), hasKey: false, models: [], usage: {} };
}
