/* 강해 시리즈 — 연속 강해를 묶고, 어디까지 왔는지 · 다음 본문은 어디인지 */
'use strict';

Screens.series = (() => {
  let sel = '';

  function stat(x) {
    const list = S.data.sermons.filter((s) => s.seriesId === x.id).sort((a, b) => a.date.localeCompare(b.date));
    const done = list.filter((s) => s.status === '설교함').length;
    return { list, done, count: list.length };
  }

  function render(el, q) {
    if (q.id) sel = q.id;
    const active = S.data.series.filter((x) => !x.done);
    const finished = S.data.series.filter((x) => x.done);
    if (!sel || !S.data.seriesById[sel]) sel = (active[0] || finished[0] || {}).id || '';
    const card = (x) => {
      const st = stat(x);
      const pct = x.total ? Math.min(100, Math.round(st.done / x.total * 100)) : 0;
      const first = st.list[0];
      return `<button type="button" class="card" data-sel="${x.id}" style="text-align:left;color:var(--text);${x.id === sel ? 'border-color:var(--brand);box-shadow:0 0 0 1px var(--brand)' : ''}">
        <div class="row" style="gap:8px;font-size:12px;font-weight:700;color:var(--muted)"><span class="sq" style="background:${x.color || 'var(--brand)'}"></span>${esc(x.kind || '')}${first ? ' · ' + first._d.getFullYear() + '년 ' + (first._d.getMonth() + 1) + '월부터' : x.start ? ' · ' + esc(fmtDate(x.start)) + '부터' : ''}</div>
        <div style="font-size:18px;font-weight:800">${esc(x.name)}</div>
        <div class="row" style="align-items:baseline;gap:6px"><span style="font-size:26px;font-weight:800">${st.done}</span><span class="muted" style="font-size:13px">${x.total ? '/ ' + x.total + '편 예정' : '편 · 끝 정하지 않음'}${st.count > st.done ? ' · 준비 ' + (st.count - st.done) : ''}</span></div>
        ${x.total ? `<div class="progress"><i style="width:${pct}%;background:${x.color || 'var(--brand)'}"></i></div>` : ''}
        <div class="muted" style="font-size:13px">다음 본문 <b style="color:var(--text)">${esc(x.nextRef || '정하지 않음')}</b></div></button>`;
    };
    el.innerHTML = `
      <div class="head"><div class="t"><h1>강해 시리즈</h1><p class="sub">연속 강해를 묶어 두면 몇 번째인지, 다음 본문이 어디인지 알려 드려요. 설교 달력의 반복 일정과 이어 둘 수 있어요.</p></div>
        <button type="button" class="btn brand" data-act="new">${svgIcon(IC.plus)}새 시리즈</button></div>
      ${active.length ? `<div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:12px">${active.map(card).join('')}</div>`
        : '<div class="empty">진행 중인 강해 시리즈가 없어요. ‘새 시리즈’로 시작해 보세요. (예: 로마서 강해, 하이델베르크 요리문답 강해)</div>'}
      ${finished.length ? `<div class="row wrap" style="gap:8px;font-size:13px"><span class="muted" style="padding:6px 2px">마친 시리즈</span>${finished.map((x) => `<button type="button" class="chip" data-sel="${x.id}" style="${x.id === sel ? 'box-shadow:0 0 0 2px var(--brand)' : ''}">${esc(x.name)} · ${stat(x).count}편</button>`).join('')}</div>` : ''}
      <div id="serdetail"></div>${footer()}`;
    detail();
    el.onclick = (e) => {
      const s = e.target.closest('[data-sel]'); if (s) { sel = s.dataset.sel; render(el, {}); return; }
      const a = e.target.closest('[data-act]'); if (!a) return;
      if (a.dataset.act === 'new') edit(null);
      if (a.dataset.act === 'edit') edit(S.data.seriesById[sel]);
    };
  }

  function detail() {
    const box = $('#serdetail');
    const x = S.data.seriesById[sel];
    if (!box || !x) return;
    const st = stat(x);
    box.innerHTML = `<section class="card">
      <div class="row wrap" style="gap:10px"><span class="sq" style="background:${x.color || 'var(--brand)'}"></span><h2 class="grow" style="font-size:18px">${esc(x.name)}</h2>
        <a class="btn sm brand" href="#sermon?new=1&series=${x.id}&kind=${encodeURIComponent(x.kind || '주일')}&ref=${encodeURIComponent(x.nextRef || '')}">${svgIcon(IC.plus)}다음 설교 기록</a>
        <button type="button" class="btn sm" data-act="edit">시리즈 고치기</button></div>
      ${x.note ? `<p class="muted" style="margin:0;font-size:13px;line-height:1.6">${esc(x.note)}</p>` : ''}
      ${st.list.length ? `<table class="simple"><thead><tr><th style="width:52px">차례</th><th>날짜</th><th>본문</th><th>제목</th><th>상태</th></tr></thead><tbody>
        ${st.list.map((s, i) => `<tr style="cursor:pointer" data-go="${s.id}"><td class="num"><b>${s.seriesNo || i + 1}</b></td><td class="num nowrap">${fmtDate(s.date, true)}</td><td><b>${esc(s._ref || '본문 미정')}</b></td><td>${esc(s.title || '제목 미정')}</td><td>${statusHtml(s.status)}</td></tr>`).join('')}
      </tbody></table>` : '<div class="empty">아직 이 시리즈로 기록한 설교가 없어요. 위의 ‘다음 설교 기록’으로 시작해 보세요.</div>'}
    </section>`;
    box.querySelectorAll('[data-go]').forEach((tr) => tr.onclick = () => go('sermon', { id: tr.dataset.go }));
  }

  async function edit(x) {
    const v = x ? { ...x } : { name: '', kind: '주일', color: SERIES_COLORS[S.data.series.length % SERIES_COLORS.length], total: 0, nextRef: '', start: iso(today()), done: false, note: '' };
    const r = await modal({
      title: x ? '시리즈 고치기' : '새 강해 시리즈',
      body: `<label class="field">이름<input class="input" id="s-name" value="${esc(v.name)}" placeholder="예: 로마서 강해"></label>
        <div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px">
          <label class="field">설교 종류<select class="select" id="s-kind">${KINDS.map((k) => `<option ${k === v.kind ? 'selected' : ''}>${k}</option>`).join('')}</select></label>
          <label class="field">예정 편 수 (모르면 비워 두세요)<input class="input" id="s-total" type="number" min="0" value="${v.total || ''}"></label>
          <label class="field">시작한 날<input class="input" id="s-start" type="date" value="${esc(v.start || '')}"></label>
          <label class="field">다음 본문<input class="input" id="s-next" value="${esc(v.nextRef || '')}" placeholder="예: 롬 8:31-39"></label>
        </div>
        <div class="field">색<div class="row wrap" id="s-colors" style="gap:8px">${SERIES_COLORS.map((c) => `<button type="button" data-c="${c}" aria-label="${c}" style="width:28px;height:28px;border-radius:8px;border:0;background:${c};box-shadow:${c === v.color ? '0 0 0 2px var(--panel),0 0 0 4px var(--text)' : 'none'}"></button>`).join('')}</div></div>
        <label class="field">메모<textarea class="textarea" id="s-note" rows="2" style="min-height:60px">${esc(v.note || '')}</textarea></label>
        <label class="check"><input type="checkbox" id="s-done" ${v.done ? 'checked' : ''}>이 시리즈를 마쳤어요</label>`,
      buttons: [...(x ? [{ label: '지우기', kind: 'danger ghost', value: 'delete' }] : []), { label: '취소', value: null }, {
        label: '저장', kind: 'brand', value: (m) => ({ ...v, name: $('#s-name', m).value.trim(), kind: $('#s-kind', m).value, total: +$('#s-total', m).value || 0,
          start: $('#s-start', m).value, nextRef: $('#s-next', m).value.trim(), note: $('#s-note', m).value.trim(), done: $('#s-done', m).checked, color: v.color }),
        check: (m) => { if (!$('#s-name', m).value.trim()) { toast('시리즈 이름을 적어 주세요', { err: true }); return false; } return true; },
      }],
      onOpen: (m) => { $('#s-colors', m).onclick = (e) => { const b = e.target.closest('[data-c]'); if (!b) return; v.color = b.dataset.c; $$('#s-colors button', m).forEach((y) => y.style.boxShadow = y === b ? '0 0 0 2px var(--panel),0 0 0 4px var(--text)' : 'none'); }; },
    });
    try {
      if (r === 'delete') {
        if (!(await confirmBox('시리즈를 지울까요?', '이 시리즈로 기록한 설교는 그대로 남고, 시리즈 표시만 빠져요.', '지우기', true))) return;
        await api('/api/series/save', { delete: x.id }); sel = ''; await loadData(); refresh(); toast('시리즈를 지웠어요'); return;
      }
      if (!r) return;
      const res = await api('/api/series/save', { series: r });
      sel = res.series.id; await loadData(); refresh(); toast('시리즈를 저장했어요');
    } catch (e) { fail(e); }
  }

  return { render: (el, q) => render(el, q || {}) };
})();
