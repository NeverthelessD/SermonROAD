/* 시작 · 새 소식 확인 · 단축키 */
'use strict';

async function boot() {
  try {
    const [settings, bible] = await Promise.all([api('/api/settings'), api('/api/bible')]);
    S.settings = settings; S.books = bible.books;
    applyTheme();
    await loadData();
    api('/api/claude').then((c) => { S.claude = c; }).catch(() => {});
    api('/api/library').then((l) => { S.lib = l; S.libRev = l.rev; }).catch(() => {});
  } catch (e) {
    $('#main').innerHTML = `<div class="page"><div class="empty">앱을 여는 중에 문제가 생겼어요: ${esc(e.message)}<br>앱을 다시 실행해 주세요.</div></div>`;
    return;
  }
  window.addEventListener('hashchange', () => render());
  await render();
  updateBadge();
  poll();
}

let pollFails = 0;
async function poll() {
  try {
    const p = await api('/api/poll');
    pollFails = 0;
    $('#offline').hidden = true;
    S.update = p.update;
    updateBadge();
    if (p.rev !== S.rev) { await loadData(); refresh(); }
    if (p.libRev !== S.libRev) {
      S.libRev = p.libRev;
      S.lib = await api('/api/library');
      if (S.route.name === 'helper' && Screens.helper.refreshLib) Screens.helper.refreshLib();
    }
  } catch (e) {
    pollFails++;
    if (pollFails >= 2) $('#offline').hidden = false;
  }
  setTimeout(poll, pollFails ? 1500 : 2500);
}

function updateBadge() {
  const b = $('#ver');
  const u = S.update || {};
  const v = S.settings.build || '';
  b.classList.toggle('attn', u.status === 'ready' || u.status === 'available');
  b.textContent = u.status === 'ready' ? '새 버전 준비됨 · 다시 시작' : u.status === 'downloading' ? '새 버전 받는 중…' : u.status === 'available' ? '새 버전이 있어요' : '버전 ' + v + (u.status === 'uptodate' ? ' · 최신' : '');
  b.onclick = async () => {
    if (u.status === 'ready') {
      if (await confirmBox('새 버전으로 다시 시작할까요?', '고치던 내용이 있다면 먼저 저장해 주세요.', '다시 시작')) api('/api/update/restart', {}).catch(fail);
    } else go('settings');
  };
}

document.addEventListener('keydown', (e) => {
  if (!(e.metaKey || e.ctrlKey) || e.altKey) return;
  const k = e.key.toLowerCase();
  if (document.querySelector('#modal-root .back')) return;
  if (k === 's') { e.preventDefault(); if (S.screen && S.screen.save) S.screen.save(); }
  if (k === 'n') { e.preventDefault(); go('sermon', { new: 1 }); }
  if (k === 'f') {
    e.preventDefault();
    const q = $('#q');
    if (q) { q.focus(); q.select(); } else { go('records'); setTimeout(() => { const x = $('#q'); if (x) x.focus(); }, 200); }
  }
});

boot();
