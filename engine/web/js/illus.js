/* 예화 보관함 — 예화 · 인용을 모아 두고, 어느 설교 · 어느 회중에게 썼는지 */
'use strict';

Screens.illus = (() => {
  let sel = '', q = '', E = null, dirty = false;

  function uses(id) { return S.data.sermons.filter((s) => (s.illus || []).includes(id)).sort((a, b) => b.date.localeCompare(a.date)); }

  function render(el, qq) {
    if (qq.id) sel = qq.id;
    const list = S.data.illus.filter((x) => !q || (x.title + ' ' + x.body + ' ' + x.source + ' ' + (x.tags || []).join(' ')).toLowerCase().includes(q.toLowerCase()))
      .sort((a, b) => (b.updated || '').localeCompare(a.updated || ''));
    if (sel !== 'new' && !S.data.illusById[sel]) sel = (list[0] || {}).id || '';
    if (!E || E.id !== (sel === 'new' ? '' : sel)) { E = sel === 'new' ? { id: '', title: '', body: '', source: '', tags: [] } : sel ? { ...S.data.illusById[sel] } : null; dirty = false; }
    el.innerHTML = `
      <div class="head"><div class="t"><h1>예화 보관함</h1><p class="sub">예화와 인용(칼빈, 스펄전, 로이드 존스 …)을 모아 두세요. 설교에 쓰면 같은 회중에게 또 쓰지 않도록 알려 드려요.</p></div>
        <label class="search" style="flex:0 1 280px">${svgIcon(IC.search)}<input id="iq" type="search" value="${esc(q)}" placeholder="예화 찾기"></label>
        <button type="button" class="btn brand" data-act="new">${svgIcon(IC.plus)}새 예화</button></div>
      <div class="split">
        <section class="aside" style="flex:1 1 320px" aria-label="예화 목록">
          ${list.length ? list.map((x) => {
            const u = uses(x.id);
            return `<button type="button" class="sitem" data-sel="${x.id}" style="${x.id === sel ? 'border-color:var(--brand);box-shadow:0 0 0 1px var(--brand)' : ''}">
              <div class="title">${esc(x.title)}</div>
              <div class="muted ellipsis" style="font-size:12px">${esc((x.body || '').replace(/\s+/g, ' ').slice(0, 90) || '내용 없음')}</div>
              <div class="info">${(x.tags || []).map((t) => `<span>#${esc(t)}</span>`).join('')}<span>${u.length ? u.length + '번 씀 · 마지막 ' + fmtDate(u[0].date) : '아직 쓰지 않음'}</span></div></button>`;
          }).join('') : `<div class="empty">${q ? '찾는 예화가 없어요.' : '아직 예화가 없어요. ‘새 예화’로 첫 예화를 넣어 보세요.'}</div>`}
        </section>
        <section class="main" style="flex:999 1 520px" id="illusedit"></section>
      </div>${footer()}`;
    drawEdit();
    $('#iq', el).oninput = debounce((e) => { q = e.target.value.trim(); const p = e.target.selectionStart; render(el, {}); const i = $('#iq', el); i.focus(); i.setSelectionRange(p, p); }, 200);
    el.onclick = async (e) => {
      const s = e.target.closest('[data-sel]'); if (s) { if (await canLeave()) { sel = s.dataset.sel; E = null; render(el, {}); } return; }
      const a = e.target.closest('[data-act]'); if (!a) return;
      if (a.dataset.act === 'new') { if (await canLeave()) { sel = 'new'; E = null; render(el, {}); setTimeout(() => $('#i-title') && $('#i-title').focus(), 30); } }
      if (a.dataset.act === 'save') save(el);
      if (a.dataset.act === 'delete') del(el);
    };
  }

  function drawEdit() {
    const box = $('#illusedit');
    if (!box) return;
    if (!E) { box.innerHTML = '<div class="empty">왼쪽에서 예화를 고르거나 새로 만드세요.</div>'; return; }
    const u = E.id ? uses(E.id) : [];
    box.innerHTML = `<section class="card">
      <label class="field">제목<input class="input big" id="i-title" value="${esc(E.title)}" placeholder="예: 요셉의 고백 (창 50:20)"></label>
      <label class="field">내용<textarea class="textarea serif" id="i-body" rows="8">${esc(E.body)}</textarea></label>
      <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px">
        <label class="field">출처<input class="input" id="i-src" value="${esc(E.source)}" placeholder="예: 스펄전, 『아침 묵상』"></label>
        <label class="field">꼬리표 (쉼표로)<input class="input" id="i-tags" value="${esc((E.tags || []).join(', '))}" placeholder="예: 섭리, 고난, 위로"></label>
      </div>
      <div class="row" style="justify-content:flex-end;gap:8px">${E.id ? `<button type="button" class="btn ghost danger" data-act="delete">${svgIcon(IC.trash)}지우기</button>` : ''}<button type="button" class="btn brand" data-act="save">저장</button></div>
    </section>
    ${E.id ? `<section class="card"><h2>쓴 설교 <span class="muted" style="font-weight:600">${u.length}번</span></h2>
      ${u.length ? `<div class="slist">${u.map((s) => sermonItem(s)).join('')}</div>` : '<div class="empty">아직 이 예화를 쓴 설교가 없어요. 설교 기록 › 쓴 예화에서 고를 수 있어요.</div>'}</section>` : ''}`;
    ['i-title', 'i-body', 'i-src', 'i-tags'].forEach((id) => { $('#' + id, box).oninput = () => { dirty = true; }; });
  }

  function read() {
    return { ...E, title: $('#i-title').value.trim(), body: $('#i-body').value, source: $('#i-src').value.trim(), tags: $('#i-tags').value.split(/[,，]/).map((t) => t.trim().replace(/^#/, '')).filter(Boolean) };
  }
  async function save(el) {
    const v = read();
    if (!v.title) { toast('제목을 적어 주세요', { err: true }); return false; }
    try { const r = await api('/api/illus/save', { illus: v }); dirty = false; sel = r.illus.id; E = null; await loadData(); render(el, {}); toast('예화를 저장했어요'); return true; } catch (e) { fail(e); return false; }
  }
  async function del(el) {
    const u = uses(E.id);
    if (!(await confirmBox('이 예화를 지울까요?', u.length ? `이 예화를 쓴 설교 ${u.length}편에서도 빠져요.` : '보관함에서 지워요.', '지우기', true))) return;
    try { await api('/api/illus/save', { delete: E.id }); dirty = false; sel = ''; E = null; await loadData(); render(el, {}); toast('예화를 지웠어요'); } catch (e) { fail(e); }
  }
  async function canLeave() {
    if (!dirty) return true;
    const r = await modal({ title: '저장하지 않은 예화가 있어요', body: '<p style="margin:0;color:var(--muted)">고친 내용을 저장할까요?</p>',
      buttons: [{ label: '취소', value: 'stay' }, { label: '저장하지 않기', kind: 'danger ghost', value: 'drop' }, { label: '저장', kind: 'brand', value: 'save' }] });
    if (r === 'save') return await save($('#main .page'));
    if (r === 'drop') { dirty = false; return true; }
    return false;
  }

  return { render: (el, q) => render(el, q || {}), canLeave, refresh: () => { if (!dirty) { const el = $('#main .page'); E = null; render(el, {}); } }, save: () => { if (E) save($('#main .page')); } };
})();
