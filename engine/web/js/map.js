/* 설교 지도 — 66권을 한눈에: 많이 설교한 책일수록 칸 색이 진해요 */
'use strict';

Screens.map = (() => {
  let selBook = 0, selCh = 0;

  function counts() {
    const book = {}, chap = {};
    for (const s of S.data.sermons) {
      const seenB = new Set(), seenC = new Set();
      for (const p of s.passages || []) {
        if (!(p.book > 0)) continue;
        if (!seenB.has(p.book)) { seenB.add(p.book); book[p.book] = (book[p.book] || 0) + 1; }
        for (let c = p.c1; c <= (p.c2 || p.c1); c++) {
          const k = p.book + ':' + c;
          if (!seenC.has(k)) { seenC.add(k); chap[k] = (chap[k] || 0) + 1; }
        }
      }
    }
    return { book, chap };
  }
  const level = (n) => n >= 20 ? 1 : n >= 10 ? 0.75 : n >= 5 ? 0.5 : n > 0 ? 0.3 : 0;
  function paint(color, n) {
    const l = level(n);
    if (!l) return '';
    return `background:${l >= 1 ? color : hexA(color, l)};color:${l >= 0.75 ? '#14151C' : 'var(--text)'}`;
  }

  function render(el) {
    const C = counts();
    const sermons = S.data.sermons;
    let ot = 0, nt = 0;
    for (const s of sermons) for (const p of s.passages || []) { if (p.book > 0) { if (p.book <= 39) ot++; else nt++; } }
    const covered = S.books.filter((b) => C.book[b.n]).length;
    const untouched = S.books.filter((b) => !C.book[b.n]).map((b) => b.name);
    if (!selBook) {
      let best = 0, bn = 0;
      for (const [k, v] of Object.entries(C.book)) if (v > best) { best = v; bn = +k; }
      selBook = bn || 45;
    }
    const pct = ot + nt ? Math.round(ot / (ot + nt) * 100) : 0;
    const groups = (otFlag) => {
      const out = [];
      for (const b of S.books.filter((x) => x.ot === otFlag)) {
        let g = out.find((x) => x.name === b.group);
        if (!g) { g = { name: b.group, color: bookColor(b), books: [] }; out.push(g); }
        g.books.push(b);
      }
      return out;
    };
    const tsec = (name, info, otFlag) => `<div class="card" style="gap:10px">
      <div class="row" style="align-items:baseline;gap:10px"><h2 style="font-size:16px">${name}</h2><span class="muted" style="font-size:13px">${info}</span></div>
      ${groups(otFlag).map((g) => `<div class="map-group"><div class="gname"><span class="sq" style="background:${g.color}"></span>${esc(g.name)}</div>
        <div class="books">${g.books.map((b) => {
          const n = C.book[b.n] || 0;
          return `<button type="button" class="book ${n ? 'has' : ''} ${b.n === selBook ? 'on' : ''}" data-book="${b.n}" title="${esc(b.name)} · 설교 ${n}편" aria-label="${esc(b.name)} 설교 ${n}편" style="${paint(g.color, n)}"><b>${esc(b.abbr)}</b><span>${n ? n + '편' : '–'}</span></button>`;
        }).join('')}</div></div>`).join('')}</div>`;

    const up = typeof upcomingEvents === 'function' ? upcomingEvents(today(), 14).slice(0, 3) : [];
    el.innerHTML = `
      <div class="head">
        <div class="t"><h1>설교 지도</h1><p class="sub">칸을 누르면 그 책의 장별 설교가 오른쪽에 열려요. 색이 진할수록 많이 설교한 책이에요.</p></div>
        <label class="search" style="flex:0 1 300px">${svgIcon(IC.search)}<input id="q" type="search" placeholder="본문 · 제목 · 내용 · 장소 찾기" aria-label="찾기"><span class="kbd">⌘F</span></label>
        <a class="btn brand" href="#sermon?new=1">${svgIcon(IC.plus)}새 설교</a>
      </div>
      <div class="row wrap" style="gap:10px">
        <div class="stat">기록한 설교 <b>${sermons.length}편</b></div>
        <div class="stat">설교한 책 <b>${covered} / 66권</b></div>
        <div class="stat">구약 <b style="font-size:14px;margin:0">${pct}%</b> · 신약 <b style="font-size:14px;margin:0">${ot + nt ? 100 - pct : 0}%</b></div>
        <div class="stat" style="flex:1 1 320px;border-style:dashed;background:transparent">아직 설교하지 않은 책 ${untouched.length}권${untouched.length ? ' · <span style="color:var(--text)">' + esc(untouched.slice(0, 6).join(' · ')) + (untouched.length > 6 ? ' 외 ' + (untouched.length - 6) + '권' : '') + '</span>' : ''}</div>
      </div>
      <a class="upcoming" href="#calendar">
        <span class="lab">${svgIcon(IC.cal)}다가오는 설교</span>
        ${up.length ? up.map((e) => `<span style="font-size:14px"><b>${esc(upLabel(e))}</b> <span class="muted">${esc(kindLabel(e))} · ${esc(e.ref || e.series || '본문 미정')} · ${esc(e.status)}</span></span>`).join('')
          : '<span class="muted" style="font-size:14px">앞으로 2주 안에 잡힌 설교가 없어요. 설교 달력에서 반복 일정을 정해 두면 저절로 채워져요.</span>'}
        <span class="muted" style="margin-left:auto;font-size:13px;font-weight:700">설교 달력 ›</span>
      </a>
      <div class="split">
        <section class="main" aria-label="66권 지도">
          ${tsec('구약', '39권', true)}${tsec('신약', '27권', false)}
          <div class="legend"><span>진하기</span>
            <span><i style="border:1px dashed var(--line2)"></i>아직 없음</span><span><i style="background:#9B6BF24D"></i>1–4편</span>
            <span><i style="background:#9B6BF280"></i>5–9편</span><span><i style="background:#9B6BF2BF"></i>10–19편</span><span><i style="background:#9B6BF2"></i>20편 이상</span></div>
        </section>
        <aside class="aside" id="bookside"></aside>
      </div>${footer()}`;
    renderSide(C);
    el.addEventListener('click', (e) => {
      const b = e.target.closest('[data-book]');
      if (b) { selBook = +b.dataset.book; selCh = 0; $$('.book', el).forEach((x) => x.classList.toggle('on', x === b)); renderSide(counts()); }
    });
    $('#q', el).addEventListener('keydown', (e) => { if (e.key === 'Enter' && e.target.value.trim()) go('records', { q: e.target.value.trim() }); });
  }

  function upLabel(e) {
    const d = parseISO(e.date);
    const t = today();
    const diff = Math.round((d - t) / 86400000);
    const when = diff === 0 ? '오늘' : diff === 1 ? '내일' : (d.getMonth() + 1) + '.' + d.getDate();
    return when + ' ' + DOW[d.getDay()] + (e.time ? ' ' + e.time : '');
  }

  function renderSide(C) {
    const side = $('#bookside');
    if (!side) return;
    const b = bookOf(selBook);
    const color = bookColor(b);
    const n = C.book[b.n] || 0;
    let chCovered = 0;
    const cells = [];
    for (let c = 1; c <= b.ch; c++) {
      const v = C.chap[b.n + ':' + c] || 0;
      if (v) chCovered++;
      cells.push(`<button type="button" class="chap ${v ? 'has' : ''} ${c === selCh ? 'on' : ''}" data-ch="${c}" aria-label="${c}${b.n === 19 ? '편' : '장'} 설교 ${v}편" title="${c}${b.n === 19 ? '편' : '장'} · ${v}편" style="${paint(color, v * 5)}">${c}</button>`);
    }
    const list = S.data.sermons.filter((s) => (s.passages || []).some((p) => p.book === b.n && (!selCh || (p.c1 <= selCh && (p.c2 || p.c1) >= selCh)))).slice().reverse();
    const unit = b.n === 19 ? '편' : '장';
    const refNew = selCh ? `${b.abbr} ${selCh}${unit}` : b.name;
    side.innerHTML = `<div class="card">
      <div class="row" style="align-items:flex-start">
        <div class="grow" style="min-width:0">
          <div class="row" style="gap:6px;font-size:12px;font-weight:700;color:var(--muted)"><span class="sq" style="background:${color}"></span>${b.ot ? '구약' : '신약'} · ${esc(b.group)}</div>
          <h2 style="margin-top:4px;font-size:24px">${esc(b.name)}</h2>
          <div class="muted" style="margin-top:4px;font-size:13px">설교 ${n}편 · ${b.ch}${unit} 중 ${chCovered}${unit} 설교함</div>
        </div>
        <a class="btn xs" href="${bibleURL({ book: b.n, c1: selCh || 1 })}" target="_blank" title="HOLYBIBLE에서 열기">HOLYBIBLE ${svgIcon(IC.ext, 12)}</a>
      </div>
      <div><div class="muted" style="font-size:12px;font-weight:700;margin-bottom:8px">${unit}별 설교 <span class="faint" style="font-weight:600">· 다시 누르면 전체</span></div><div class="chaps">${cells.join('')}</div></div>
      <div class="hr"></div>
      <div class="row" style="justify-content:space-between"><h3 style="font-size:15px">${esc(b.name)} ${selCh ? selCh + unit + ' ' : ''}설교 <span class="muted" style="font-weight:600">${list.length}편</span></h3>
        <a class="link" href="#sermon?new=1&ref=${encodeURIComponent(refNew)}">+ ${selCh ? '이 ' + unit + '으로' : '이 책으로'} 새 설교</a></div>
      ${list.length ? `<div class="slist">${list.slice(0, 40).map((s) => sermonItem(s)).join('')}</div>${list.length > 40 ? `<a class="link" href="#records?q=${encodeURIComponent(b.abbr)}">모두 보기 (${list.length}편)</a>` : ''}`
        : `<div class="empty">${selCh ? '아직 이 ' + unit + '으로 한 설교가 없어요.' : '아직 이 책으로 한 설교가 없어요.'} 위의 ‘새 설교’로 바로 기록할 수 있어요.</div>`}
    </div>`;
    side.onclick = (e) => {
      const c = e.target.closest('[data-ch]');
      if (c) { const v = +c.dataset.ch; selCh = selCh === v ? 0 : v; renderSide(counts()); }
    };
  }

  return { render };
})();

// 설교 한 편을 보여 주는 칸 (지도 · 요리문답 · 시리즈에서 같이 써요)
function sermonItem(s, opts = {}) {
  const ser = S.data.seriesById[s.seriesId];
  const dupNote = opts.note ? `<div class="warn-text">${esc(opts.note)}</div>` : '';
  return `<a class="sitem" href="#sermon?id=${s.id}">
    <div class="meta"><span class="num">${fmtDate(s.date)}</span>${kindChip(s.kind, kindLabel(s))}${s.season ? `<span class="season">${esc(s.season)}</span>` : ''}${s.status !== '설교함' ? statusHtml(s.status) : ''}</div>
    <div class="title">${esc(s.title || '제목 미정')}</div>
    <div class="info"><b>${esc(s._ref || '본문 미정')}</b><span>${esc(s.place || '')}</span>${ser ? `<span>${esc(ser.name)}${s.seriesNo ? ' ' + s.seriesNo : ''}</span>` : ''}</div>${dupNote}
  </a>`;
}
