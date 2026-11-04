/* 설교 기록 — 기간 › 연도 › 월, 세 단계로 돌아보기 · 찾기 */
'use strict';

Screens.records = (() => {
  let P = null, Y = 0, M = 0;

  function periods() {
    const base = [[2013, 2015], [2016, 2020], [2021, 2025], [2026, 2030], [2031, 2035], [2036, 2040]];
    const years = S.data.sermons.map((s) => s._d && s._d.getFullYear()).filter(Boolean);
    const min = years.length ? Math.min(...years) : 2013, max = years.length ? Math.max(...years) : 2040;
    const out = [];
    if (min < 2013) out.push([min, 2012]);
    out.push(...base);
    if (max > 2040) out.push([2041, max]);
    return out;
  }
  const countYear = (y) => S.data.sermons.filter((s) => s._d && s._d.getFullYear() === y).length;
  const monthList = (y, m) => S.data.sermons.filter((s) => s._d && s._d.getFullYear() === y && s._d.getMonth() + 1 === m);
  function latestMonth(y) {
    const t = today();
    for (let m = 12; m >= 1; m--) if (monthList(y, m).length && (y < t.getFullYear() || m <= t.getMonth() + 1)) return m;
    return y === t.getFullYear() ? t.getMonth() + 1 : 1;
  }
  function pickPeriod(i) {
    const ps = periods(); P = i;
    const [a, b] = ps[i];
    let y = a;
    for (let k = Math.min(b, today().getFullYear()); k >= a; k--) if (countYear(k)) { y = k; break; }
    if (!countYear(y) && today().getFullYear() >= a && today().getFullYear() <= b) y = today().getFullYear();
    Y = y; M = latestMonth(y);
  }

  function render(el, q) {
    if (q.q) return renderSearch(el, q.q);
    const ps = periods();
    const t = today();
    if (q.y) { Y = +q.y; M = +(q.m || latestMonth(Y)); P = ps.findIndex(([a, b]) => Y >= a && Y <= b); }
    if (P === null || P < 0 || P >= ps.length) pickPeriod(Math.max(0, ps.findIndex(([a, b]) => t.getFullYear() >= a && t.getFullYear() <= b)));
    const [pa, pb] = ps[P];
    const maxN = Math.max(1, ...Array.from({ length: 2041 - 2000 }, (_, i) => countYear(2000 + i)));
    const periodsHtml = ps.map(([a, b], i) => {
      const total = Array.from({ length: b - a + 1 }, (_, k) => countYear(a + k)).reduce((x, y) => x + y, 0);
      const future = a > t.getFullYear();
      const bars = Array.from({ length: b - a + 1 }, (_, k) => `<i style="height:${Math.round(countYear(a + k) / maxN * 100)}%"></i>`).join('');
      return `<button type="button" class="pbtn ${i === P ? 'on' : ''} ${future && !total ? 'dim' : ''}" data-p="${i}" aria-pressed="${i === P}">
        <span class="row" style="justify-content:space-between;align-items:baseline"><b class="num" style="font-size:15px">${a}–${String(b).slice(2)}</b><span class="muted" style="font-size:12px;font-weight:700">${total ? total + '편' : future ? '아직' : '0편'}</span></span>
        <span class="bars">${bars}</span></button>`;
    }).join('');
    const yearsHtml = Array.from({ length: pb - pa + 1 }, (_, k) => {
      const yy = pa + k, n = countYear(yy);
      return `<button type="button" class="ybtn ${yy === Y ? 'on' : ''} ${!n && yy > t.getFullYear() ? 'dim' : ''}" data-y="${yy}" aria-pressed="${yy === Y}"><b class="num" style="font-size:16px">${yy}년</b><span class="muted" style="font-size:12px;font-weight:700">${n ? n + '편' : yy > t.getFullYear() ? '아직' : '0편'}</span></button>`;
    }).join('');
    const kindCount = {};
    const monthsHtml = Array.from({ length: 12 }, (_, i) => {
      const mm = i + 1, list = monthList(Y, mm);
      list.forEach((s) => { kindCount[s.kind] = (kindCount[s.kind] || 0) + 1; });
      const future = Y > t.getFullYear() || (Y === t.getFullYear() && mm > t.getMonth() + 1);
      const planned = future || (Y === t.getFullYear() && mm === t.getMonth() + 1) ? list.filter((s) => s.status !== '설교함').length : 0;
      return `<button type="button" class="mbtn ${mm === M ? 'on' : ''} ${future && !list.length ? 'dim' : ''}" data-m="${mm}" aria-label="${mm}월 설교 ${list.length}편" aria-pressed="${mm === M}">
        <span class="row" style="justify-content:space-between;align-items:baseline;gap:4px"><b style="font-size:15px">${mm}월</b><span class="muted" style="font-size:12px;font-weight:700">${list.length ? list.length + '편' : future ? '아직' : '0편'}${planned ? ' · 예정 ' + planned : ''}</span></span>
        <span class="kb">${list.length ? list.map((s) => `<i style="background:${KC[s.kind]}"></i>`).join('') : '<i style="background:var(--panel2)"></i>'}</span></button>`;
    }).join('');
    const yearKinds = KINDS.filter((k) => kindCount[k]).map((k) => `<span class="chip sm" style="background:var(--panel);border:1px solid var(--line);font-weight:600"><span class="dot" style="background:${KC[k]}"></span>${k} <b>${kindCount[k]}</b></span>`).join('');

    el.innerHTML = `
      <div class="head">
        <div class="t"><h1>설교 기록</h1>
          <nav class="crumb row wrap" aria-label="위치" style="margin-top:8px;gap:6px;font-size:14px">
            <button type="button" class="link" data-crumb="all" style="color:var(--muted);font-size:14px">전체</button><span>›</span>
            <button type="button" class="link" data-crumb="p" style="color:var(--muted);font-size:14px">${pa}–${pb}년</button><span>›</span>
            <span>${Y}년</span><span>›</span><b style="color:var(--text)">${M}월</b></nav></div>
        <label class="search" style="flex:0 1 280px">${svgIcon(IC.search)}<input id="q" type="search" placeholder="본문 · 제목 · 장소 찾기" aria-label="설교 찾기"></label>
        <a class="btn brand" href="#sermon?new=1">${svgIcon(IC.plus)}새 설교</a>
      </div>
      <section class="col" aria-label="기간 고르기"><div class="step"><span class="k">1</span><h3>기간</h3></div>
        <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:8px">${periodsHtml}</div></section>
      <section class="col" aria-label="연도 고르기"><div class="step"><span class="k">2</span><h3>${pa}–${pb}년 중 연도</h3></div>
        <div class="row wrap">${yearsHtml}</div></section>
      <div class="split">
        <section class="main" aria-label="월 고르기" style="gap:8px">
          <div class="step"><span class="k">3</span><h3>${Y}년 달별 <span class="muted" style="font-weight:600">· 모두 ${countYear(Y)}편</span></h3></div>
          <div class="mgrid">${monthsHtml}</div>
          <div class="row wrap" style="gap:6px;padding-top:4px">${yearKinds}</div>
          <div class="row wrap" style="gap:8px;padding-top:6px">
            <button type="button" class="btn sm" data-export="year">${Y}년 엑셀로</button>
            <button type="button" class="btn sm" data-export="all">전체 엑셀로</button>
            <button type="button" class="btn sm" data-print="year">${Y}년 인쇄용 목록</button>
          </div>
        </section>
        <aside class="aside" id="monthside"></aside>
      </div>${footer()}`;
    renderMonth();
    el.onclick = (e) => {
      const p = e.target.closest('[data-p]'); if (p) { pickPeriod(+p.dataset.p); return render(el, {}); }
      const y = e.target.closest('[data-y]'); if (y) { Y = +y.dataset.y; M = latestMonth(Y); return render(el, {}); }
      const m = e.target.closest('[data-m]'); if (m) { M = +m.dataset.m; $$('.mbtn', el).forEach((x) => x.classList.toggle('on', x === m)); el.querySelector('.crumb b').textContent = M + '월'; return renderMonth(); }
      const c = e.target.closest('[data-crumb]');
      if (c) { if (c.dataset.crumb === 'all') { P = null; } else { pickPeriod(P); } return render(el, {}); }
      const x = e.target.closest('[data-export]'); if (x) return doExport(x.dataset.export);
      const pr = e.target.closest('[data-print]'); if (pr) return openPrint(pr.dataset.print);
    };
    $('#q', el).addEventListener('keydown', (e) => { if (e.key === 'Enter' && e.target.value.trim()) go('records', { q: e.target.value.trim() }); });
  }

  function renderMonth() {
    const side = $('#monthside');
    if (!side) return;
    const t = today();
    const list = monthList(Y, M);
    const future = Y > t.getFullYear() || (Y === t.getFullYear() && M > t.getMonth() + 1);
    side.innerHTML = `<div class="card">
      <div class="row" style="align-items:flex-end"><div class="grow"><div class="muted" style="font-size:12px;font-weight:700">${Y}년</div>
        <h2 style="font-size:24px;margin-top:2px">${M}월 설교 <span class="muted" style="font-size:16px">${list.length}편</span></h2></div>
        <a class="link" href="#calendar?y=${Y}&m=${M}">달력으로 보기</a></div>
      ${list.length ? `<div>${list.map((s) => rowHtml(s)).join('')}</div>`
        : `<div class="empty">${future ? '아직 오지 않은 달이에요. 예정된 설교는 설교 달력에서 볼 수 있어요.' : '이 달에는 기록한 설교가 없어요.'}</div>`}
      <div class="row wrap" style="gap:8px;padding-top:2px">
        <button type="button" class="btn sm" data-export="month" ${list.length ? '' : 'disabled'}>이 달 엑셀로 내보내기</button>
        <button type="button" class="btn sm" data-print="month" ${list.length ? '' : 'disabled'}>인쇄용 목록</button>
      </div></div>`;
  }

  function rowHtml(s, showMonth) {
    const d = s._d;
    const dow = d.getDay();
    return `<a class="mrow" href="#sermon?id=${s.id}">
      <span class="day"><b class="num" style="${showMonth ? 'font-size:14px' : ''}">${showMonth ? (d.getMonth() + 1) + '.' : ''}${d.getDate()}</b><span style="color:${dow === 0 ? 'var(--sun)' : dow === 6 ? 'var(--sat)' : 'var(--muted)'}">${DOW[dow]}</span></span>
      <span class="grow col" style="gap:4px;min-width:0">
        <span class="row wrap" style="gap:6px">${kindChip(s.kind, kindLabel(s))}${s.season ? `<span class="season" style="border:0;padding:0">${esc(s.season)}</span>` : ''}${s.status !== '설교함' ? statusHtml(s.status) : ''}</span>
        <span style="font-size:15px;font-weight:800;line-height:1.35">${esc(s.title || '제목 미정')}</span>
        <span class="row wrap" style="gap:4px 10px;font-size:12px;color:var(--muted)"><b style="color:var(--text)">${esc(s._ref || '본문 미정')}</b><span>${esc(s.place || '')}</span></span>
      </span></a>`;
  }

  function matches(s, words, passages) {
    if (passages.length && (s.passages || []).some((p) => passages.some((q) => overlap(p, q)))) return true;
    const ser = S.data.seriesById[s.seriesId];
    const hay = [s.title, s._ref, s.passageText, s.place, s.kindOther, s.season, s.thesis, s.summary, (s.points || []).join(' '), s.refs, ser && ser.name, s.date].join(' ').toLowerCase();
    return words.every((w) => hay.includes(w));
  }

  async function renderSearch(el, q) {
    let passages = [];
    try { passages = (await api('/api/parse', { text: q })).passages.filter((p) => p.book > 0); } catch (e) { /* 그냥 낱말로 찾기 */ }
    const words = q.toLowerCase().split(/\s+/).filter(Boolean);
    const list = S.data.sermons.filter((s) => matches(s, words, passages)).slice().reverse();
    const byYear = {};
    for (const s of list) (byYear[s._d ? s._d.getFullYear() : '날짜 없음'] = byYear[s._d ? s._d.getFullYear() : '날짜 없음'] || []).push(s);
    el.innerHTML = `
      <div class="head">
        <div class="t"><div class="crumb"><a href="#records">설교 기록</a> › 찾기</div><h1 style="margin-top:4px">‘${esc(q)}’ 찾기 <span class="muted" style="font-size:18px">${list.length}편</span></h1>
          ${passages.length ? `<p class="sub">본문 ${esc(passages.map((p) => p.ref).join('; '))} 과 겹치는 설교도 함께 찾았어요.</p>` : ''}</div>
        <label class="search" style="flex:0 1 300px">${svgIcon(IC.search)}<input id="q" type="search" value="${esc(q)}" aria-label="설교 찾기"></label>
        <a class="btn" href="#records">닫기</a>
      </div>
      ${list.length ? Object.keys(byYear).sort().reverse().map((y) => `<section class="card"><h2>${y}년 <span class="muted" style="font-weight:600">${byYear[y].length}편</span></h2><div>${byYear[y].map((s) => rowHtml(s, true)).join('')}</div></section>`).join('')
        : '<div class="empty">찾는 설교가 없어요. 낱말을 줄이거나 본문(예: 롬 8장)으로 찾아보세요.</div>'}
      ${footer()}`;
    const inp = $('#q', el);
    inp.focus(); inp.setSelectionRange(inp.value.length, inp.value.length);
    inp.addEventListener('keydown', (e) => { if (e.key === 'Enter') { const v = e.target.value.trim(); v ? go('records', { q: v }) : go('records'); } });
  }

  async function doExport(scope) {
    try {
      const r = await api('/api/export', { scope, y: Y, m: M });
      toast(`엑셀 파일을 저장했어요 (${r.count}편) · Finder 에서 보여 드릴게요`);
    } catch (e) { fail(e); }
  }
  function openPrint(scope) {
    openExternal(`/print?t=${encodeURIComponent(TOKEN)}&scope=${scope}&y=${Y}&m=${M}`);
    toast('인쇄용 목록을 브라우저로 열었어요. ⌘P 로 인쇄하세요');
  }

  return { render: (el, q) => render(el, q || {}) };
})();
