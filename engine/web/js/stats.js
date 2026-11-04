/* 통계 · 백업 — 해마다 · 종류별 · 구약/신약 · 장소별, 그리고 기록 지키기 */
'use strict';

Screens.stats = (() => {
  let range = 'year';
  let backups = null;

  async function render(el) {
    const t = today(), Y = t.getFullYear();
    const all = S.data.sermons;
    const inRange = all.filter((s) => s._d && (range === 'all' || s._d.getFullYear() === (range === 'year' ? Y : Y - 1)));
    const rangeLabel = range === 'all' ? '전체' : (range === 'year' ? Y : Y - 1) + '년';
    // 해마다
    const years = uniq(all.map((s) => s._d && s._d.getFullYear()).filter(Boolean)).sort();
    const ys = years.slice(-10);
    const yc = ys.map((y) => all.filter((s) => s._d && s._d.getFullYear() === y).length);
    const ymax = Math.max(1, ...yc);
    // 종류별
    const kc = KINDS.map((k) => [k, inRange.filter((s) => s.kind === k).length]);
    const kmax = Math.max(1, ...kc.map((x) => x[1]));
    // 구약 · 신약 · 책
    let ot = 0, nt = 0; const bc = {};
    for (const s of inRange) {
      const seen = new Set();
      for (const p of s.passages || []) if (p.book > 0 && !seen.has(p.book)) { seen.add(p.book); p.book <= 39 ? ot++ : nt++; bc[p.book] = (bc[p.book] || 0) + 1; }
    }
    const otp = ot + nt ? Math.round(ot / (ot + nt) * 100) : 0;
    const topBooks = Object.entries(bc).sort((a, b) => b[1] - a[1]).slice(0, 8);
    const pc = {}; for (const s of inRange) if (s.place) pc[s.place] = (pc[s.place] || 0) + 1;
    const places = Object.entries(pc).sort((a, b) => b[1] - a[1]);
    if (!backups) { try { backups = await api('/api/backups'); } catch (e) { backups = { backups: [] }; } }
    const last = (backups.backups || [])[0];
    const trash = S.data.trash || [];
    el.innerHTML = `
      <div class="head"><div class="t"><h1>통계 · 백업</h1><p class="sub">지금까지의 설교를 돌아보고, 소중한 기록을 지켜요.</p></div>
        <div class="seg" style="width:260px">${[['year', Y + '년'], ['last', (Y - 1) + '년'], ['all', '전체']].map(([k, l]) => `<button type="button" data-range="${k}" class="${range === k ? 'on' : ''}">${l}</button>`).join('')}</div></div>
      <div class="row wrap" style="gap:10px">
        <div class="stat">${rangeLabel} 설교 <b>${inRange.length}편</b></div>
        <div class="stat">설교한 책 <b>${Object.keys(bc).length}권</b></div>
        <div class="stat">설교한 곳 <b>${places.length}곳</b></div>
        <div class="stat">강해 시리즈 <b>${S.data.series.filter((x) => !x.done).length}개 진행 중</b></div>
      </div>
      <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(300px,1fr));gap:12px">
        <section class="card"><h3>해마다 설교 수</h3>
          ${ys.length ? `<div class="row" style="align-items:flex-end;gap:8px;height:160px;padding-top:10px">${ys.map((y, i) => `<div class="col" style="flex:1;align-items:center;justify-content:flex-end;gap:4px;height:100%">
            <span class="muted" style="font-size:11px;font-weight:700">${yc[i]}</span><div style="width:100%;max-width:30px;height:${Math.round(yc[i] / ymax * 78)}%;min-height:2px;border-radius:5px 5px 2px 2px;background:${y === Y ? 'var(--brand)' : 'var(--bar)'}"></div>
            <span class="faint" style="font-size:11px">${String(y).slice(2)}년</span></div>`).join('')}</div>` : '<div class="empty">아직 기록이 없어요.</div>'}
        </section>
        <section class="card" style="gap:9px"><h3>종류별 · ${rangeLabel}</h3>
          ${kc.map(([k, n]) => `<div class="row" style="gap:10px;font-size:13px"><span class="muted" style="flex:0 0 62px;font-weight:600">${k}</span>
            <div class="grow" style="height:10px;border-radius:5px;background:var(--panel2);overflow:hidden"><div style="width:${n / kmax * 100}%;height:100%;border-radius:5px;background:${KC[k]}"></div></div>
            <b style="flex:0 0 34px;text-align:right">${n}</b></div>`).join('')}
        </section>
        <section class="card"><h3>구약 · 신약 · ${rangeLabel}</h3>
          ${ot + nt ? `<div class="row" style="height:26px;border-radius:8px;overflow:hidden;font-size:12px;font-weight:800;color:#14151C;gap:0">
            <div style="width:${otp}%;height:100%;background:#36B36B;display:flex;align-items:center;padding-left:10px;white-space:nowrap">${otp >= 15 ? '구약 ' + otp + '%' : ''}</div>
            <div style="width:${100 - otp}%;height:100%;background:#4C7FF0;display:flex;align-items:center;padding-left:10px;white-space:nowrap">${100 - otp >= 15 ? '신약 ' + (100 - otp) + '%' : ''}</div></div>` : '<div class="empty">아직 기록이 없어요.</div>'}
          <h3 style="margin-top:4px">많이 설교한 책</h3>
          ${topBooks.map(([b, n]) => `<div class="row" style="justify-content:space-between;font-size:13px;padding:2px 0;border-bottom:1px solid var(--line)"><span style="font-weight:600"><span class="sq" style="background:${bookColor(bookOf(+b))};margin-right:6px"></span>${esc(bookOf(+b).name)}</span><b>${n}편</b></div>`).join('') || '<span class="faint" style="font-size:13px">—</span>'}
        </section>
        <section class="card"><h3>장소별 · ${rangeLabel}</h3>
          ${places.slice(0, 8).map(([p, n]) => `<div class="row" style="justify-content:space-between;font-size:13px;padding:2px 0;border-bottom:1px solid var(--line)"><span style="font-weight:600">${esc(p)}</span><b>${n}편</b></div>`).join('') || '<span class="faint" style="font-size:13px">—</span>'}
          ${places.length > 8 ? `<span class="faint" style="font-size:12px">그 밖에 ${places.length - 8}곳</span>` : ''}
        </section>
      </div>
      <section class="card" aria-label="백업 내보내기" style="flex-direction:row;flex-wrap:wrap;align-items:center;gap:12px">
        <div style="flex:1 1 300px;min-width:0"><h2 style="font-size:16px">백업 · 내보내기</h2>
          <p class="muted" style="margin:4px 0 0;font-size:13px;line-height:1.6">${last ? '마지막 백업 ' + esc(last.time) + ' · ' : ''}기록이 바뀐 날마다 저절로 백업해요 (최근 20개 보관). 기록은 모두 이 맥 안에만 저장돼요.</p></div>
        <button type="button" class="btn brand" data-act="backup">지금 백업</button>
        <button type="button" class="btn" data-act="restore">백업에서 되살리기</button>
        <button type="button" class="btn" data-act="export">엑셀로 내보내기</button>
        <button type="button" class="btn ghost" data-act="reveal">${svgIcon(IC.folder)}백업 폴더</button>
      </section>
      <section class="card" aria-label="휴지통"><h2>휴지통 <span class="muted" style="font-weight:600">${trash.length}편</span></h2>
        ${trash.length ? `<div class="col" style="gap:6px">${trash.slice(0, 20).map((x) => `<div class="tile row wrap" style="gap:10px;font-size:13px"><span class="num muted">${fmtDate(x.date)}</span><b class="grow">${esc(x.title || '제목 미정')}</b><span class="muted">${esc(x.ref || '')}</span><button type="button" class="btn xs" data-undel="${x.id}">되살리기</button></div>`).join('')}</div>`
          : '<span class="faint" style="font-size:13px">지운 설교가 없어요. 지운 설교는 여기에서 되살릴 수 있어요 (최근 50편).</span>'}
      </section>${footer()}`;
    el.onclick = async (e) => {
      const r = e.target.closest('[data-range]'); if (r) { range = r.dataset.range; return render(el); }
      const u = e.target.closest('[data-undel]');
      if (u) { try { await api('/api/sermon/restore', { id: u.dataset.undel }); await loadData(); render(el); toast('되살렸어요'); } catch (err) { fail(err); } return; }
      const a = e.target.closest('[data-act]'); if (!a) return;
      try {
        if (a.dataset.act === 'backup') { backups = await api('/api/backup/now', {}); toast('백업했어요'); render(el); }
        if (a.dataset.act === 'reveal') await api('/api/backup/reveal', {});
        if (a.dataset.act === 'export') { const res = await api('/api/export', { scope: 'all' }); toast(`엑셀 파일을 저장했어요 (${res.count}편) · Finder 에서 보여 드릴게요`); }
        if (a.dataset.act === 'restore') restore(el);
      } catch (err) { fail(err); }
    };
  }

  async function restore(el) {
    backups = await api('/api/backups');
    const kindName = { auto: '자동', manual: '직접', 'before-restore': '되살리기 전' };
    const list = backups.backups || [];
    const r = await modal({
      title: '백업에서 되살리기', wide: true,
      body: list.length ? `<p class="muted" style="margin:0;font-size:13px;line-height:1.6">고른 백업으로 모든 기록(설교 · 시리즈 · 예화 · 반복 일정)을 되돌려요. 지금 상태는 ‘되살리기 전’ 백업으로 먼저 남겨 둬요.</p>
        <div class="col" style="gap:6px;max-height:50vh;overflow-y:auto">${list.map((b, i) => `<label class="src"><input type="radio" name="bk" value="${esc(b.name)}" ${i === 0 ? 'checked' : ''}><span class="row grow" style="gap:10px"><b class="num">${esc(b.time)}</b><span class="chip sm">${kindName[b.kind] || b.kind}</span><span class="grow"></span><span class="faint" style="font-size:12px">${Math.max(1, Math.round(b.size / 1024))}KB</span></span></label>`).join('')}</div>`
        : '<div class="empty">아직 백업이 없어요.</div>',
      buttons: [{ label: '취소', value: null }, ...(list.length ? [{ label: '되살리기', kind: 'brand', value: (m) => (m.querySelector('input[name=bk]:checked') || {}).value }] : [])],
    });
    if (!r) return;
    if (!(await confirmBox('이 백업으로 되살릴까요?', '지금 기록은 ‘되살리기 전’ 백업으로 남겨 둬요. 언제든 다시 돌아올 수 있어요.', '되살리기'))) return;
    try { await api('/api/backup/restore', { name: r }); backups = null; await loadData(); render(el); toast('백업에서 되살렸어요'); } catch (e) { fail(e); }
  }

  return { render: (el) => render(el), refresh: () => { backups = null; const el = $('#main .page'); if (el) render(el); } };
})();
