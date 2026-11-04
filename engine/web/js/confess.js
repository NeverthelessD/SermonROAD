/* 요리문답 · 신앙고백 — 어느 주일 · 문 · 장을 설교했는지 한눈에 */
'use strict';

Screens.confess = (() => {
  let doc = 'hc';
  const selN = { hc: 1, wsc: 1, wcf: 1, dort: 1 };

  function linked(d, n) { return S.data.sermons.filter((s) => (s.confessions || []).some((c) => c.doc === d && c.n === n)); }
  function doneSet(d) {
    const set = new Set();
    for (const s of S.data.sermons) for (const c of s.confessions || []) if (c.doc === d) set.add(c.n);
    return set;
  }
  function nextOf(d) {
    const set = doneSet(d), tot = CONFESSIONS[d].total;
    const max = Math.max(0, ...set);
    if (max < tot) return max + 1;
    for (let i = 1; i <= tot; i++) if (!set.has(i)) return i;
    return 0;
  }

  function render(el, q) {
    if (q.doc && CONFESSIONS[q.doc]) doc = q.doc;
    if (q.n) selN[doc] = +q.n;
    const tabs = Object.entries(CONFESSIONS).map(([k, d]) => {
      const done = doneSet(k).size;
      return `<button type="button" class="dtab ${k === doc ? 'on' : ''}" data-doc="${k}" role="tab" aria-selected="${k === doc}">
        <span style="font-size:14px;font-weight:800">${d.name}</span>
        <span class="row" style="align-items:baseline;gap:6px"><span style="font-size:22px;font-weight:800">${done}</span><span class="muted" style="font-size:13px">/ ${d.total}${d.unit}</span></span>
        <span class="progress" style="height:6px"><i style="width:${Math.round(done / d.total * 100)}%"></i></span></button>`;
    }).join('');
    const d = CONFESSIONS[doc];
    const set = doneSet(doc), nx = nextOf(doc);
    const units = Array.from({ length: d.total }, (_, i) => {
      const n = i + 1;
      const cls = [set.has(n) ? 'done' : '', n === nx ? 'next' : '', n === selN[doc] ? 'sel' : ''].join(' ');
      const th = d.theme(n);
      return `<button type="button" class="unit ${cls}" data-n="${n}" style="min-height:${d.h}" aria-label="${d.label(n)} ${esc(th)} ${set.has(n) ? '설교함' : '아직'}"><b>${d.label(n)}</b>${th ? `<span>${esc(th)}</span>` : ''}</button>`;
    }).join('');
    const n = selN[doc];
    const list = linked(doc, n);
    const missing = Array.from({ length: Math.max(0, ...set, 0) }, (_, i) => i + 1).filter((i) => !set.has(i));
    el.innerHTML = `
      <div class="head"><div class="t"><h1>요리문답 · 신앙고백 설교</h1><p class="sub">어느 주일 · 문 · 장을 설교했는지 한눈에 봐요. 설교 기록에서 조항을 연결하면 여기에 저절로 채워져요.</p></div></div>
      <div role="tablist" aria-label="문서" style="display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:12px">${tabs}</div>
      <div class="split">
        <section class="main card" aria-label="${d.name}">
          <div class="row wrap" style="gap:14px"><h2 style="font-size:16px">${d.name}</h2>
            <span class="st"><i style="width:12px;height:12px;border-radius:4px;background:var(--conf-done)"></i>설교함</span>
            <span class="st"><i style="width:12px;height:12px;border-radius:4px;border:1px dashed var(--line2)"></i>아직</span>
            <span class="st"><i style="width:12px;height:12px;border-radius:4px;border:2px solid var(--brand)"></i>다음 차례</span></div>
          ${doc === 'wsc' ? `<div class="row wrap faint" style="gap:4px 12px;font-size:12px">${WSC_PARTS.map(([a, b, t]) => `<span>${a}–${b}문 ${esc(t)}</span>`).join('')}</div>` : ''}
          <div class="ugrid" style="grid-template-columns:repeat(auto-fill,minmax(${d.min},1fr))">${units}</div>
        </section>
        <aside class="aside"><section class="card" aria-label="고른 항목">
          <div class="muted" style="font-size:12px;font-weight:700">${d.name}</div>
          <h2 class="serif" style="font-size:24px;font-weight:700">${d.label(n)}</h2>
          <div style="font-size:15px;font-weight:700">${esc(d.theme(n) || (doc === 'wsc' ? wscPart(n) : ''))}</div>
          <div class="hr"></div>
          ${list.length ? `<div class="slist">${list.map((s) => sermonItem(s)).join('')}</div>` : '<div class="empty">아직 이 항목으로 한 설교가 없어요.</div>'}
          <a class="btn brand" href="#sermon?new=1&conf=${doc}:${n}">이 항목으로 설교 기록</a>
          ${nx ? `<div class="note" style="background:var(--brand-soft)"><b style="color:var(--brand)">다음 차례</b> · ${d.label(nx)}${d.theme(nx) ? ' · ' + esc(d.theme(nx)) : ''}${missing.length ? `<br><span class="muted">건너뛴 곳: ${missing.slice(0, 8).map((m) => d.label(m)).join(', ')}${missing.length > 8 ? ' 외' : ''}</span>` : ''}</div>`
            : '<div class="note" style="background:var(--ok-bg)"><b style="color:var(--ok)">모두 설교했어요</b></div>'}
        </section></aside>
      </div>${footer()}`;
    el.onclick = (e) => {
      const t = e.target.closest('[data-doc]'); if (t) { doc = t.dataset.doc; return render(el, {}); }
      const u = e.target.closest('[data-n]'); if (u) { selN[doc] = +u.dataset.n; return render(el, {}); }
    };
  }

  return { render: (el, q) => render(el, q || {}) };
})();
