/* 설교 달력 (스케줄러) — 한 달을 한 화면에: 설교 일정 · 준비 상태 · 절기 · 반복 일정 */
'use strict';

// 반복 일정에서 나온 '예정' 칸 (아직 기록이 없는 앞날)
function ruleEvents(from, to) {
  const out = [];
  const t0 = today();
  const start = from < t0 ? t0 : from;
  const firstRefUsed = {};
  for (const r of S.data.rules.filter((x) => x.active !== false)) {
    const d = new Date(start);
    while (d.getDay() !== r.dow) d.setDate(d.getDate() + 1);
    for (; d <= to; d.setDate(d.getDate() + 7)) {
      if (r.every === 2 && r.start) {
        const base = parseISO(r.start);
        if (base && Math.round((d - base) / (7 * 86400000)) % 2 !== 0) continue;
      }
      const date = iso(d);
      const taken = S.data.sermons.some((s) => s.date === date && s.kind === r.kind);
      if (taken) continue;
      const ser = S.data.seriesById[r.seriesId];
      let ref = '';
      if (ser && ser.nextRef && !firstRefUsed[ser.id]) {
        // 시리즈의 다음 본문은 가장 가까운 예정 칸 하나에만 넣어요
        const later = S.data.sermons.some((s) => s.seriesId === ser.id && s.date >= iso(t0) && s.date < date);
        if (!later) { ref = ser.nextRef; firstRefUsed[ser.id] = true; }
      }
      out.push({ virtual: true, rule: r.id, date, _d: new Date(d), time: r.time || '', kind: r.kind, kindOther: '', place: r.place || '',
        seriesId: r.seriesId || '', series: ser ? ser.name : (r.note || ''), ref, title: '', status: ref ? '준비 중' : '본문 미정' });
    }
  }
  return out;
}

// 다가오는 설교 (설교 지도 · 달력에서): 오늘부터 days 일 안의 기록 + 예정
function upcomingEvents(from, days) {
  const to = addDays(from, days);
  const real = S.data.sermons.filter((s) => s._d && s._d >= from && s._d <= to && s.status !== '설교함')
    .map((s) => ({ ...s, ref: s._ref, series: (S.data.seriesById[s.seriesId] || {}).name || '' }));
  const all = real.concat(ruleEvents(from, to));
  all.sort((a, b) => (a.date + (a.time || '')).localeCompare(b.date + (b.time || '')));
  return all;
}

function newSermonQuery(e) {
  return { new: 1, date: e.date, time: e.time, kind: e.kind, place: e.place, series: e.seriesId, ref: e.ref };
}

Screens.calendar = (() => {
  let y, m, selD, hide = {};

  function monthEvents() {
    const from = new Date(y, m - 1, 1), to = new Date(y, m, 0);
    const real = S.data.sermons.filter((s) => s._d && s._d >= from && s._d <= to)
      .map((s) => ({ ...s, ref: s._ref, series: (S.data.seriesById[s.seriesId] || {}).name || '' }));
    const all = real.concat(ruleEvents(from, to));
    all.sort((a, b) => (a.date + (a.time || '')).localeCompare(b.date + (b.time || '')));
    return all;
  }
  function evStyle(e) {
    const m2 = { '설교함': ['var(--done)', 'none', '.62'], '원고 완료': ['var(--ready)', 'none', '1'], '준비 중': ['var(--prep)', 'none', '1'], '본문 미정': ['transparent', '1.5px dashed var(--faint)', '1'] };
    return m2[e.status] || m2['준비 중'];
  }

  function render(el, q) {
    const t = today();
    if (q.y && q.m) { y = +q.y; m = +q.m; }
    if (!y) { y = t.getFullYear(); m = t.getMonth() + 1; }
    if (q.d) selD = +q.d;
    if (!selD) selD = (y === t.getFullYear() && m === t.getMonth() + 1) ? t.getDate() : 1;
    const evs = monthEvents();
    const vis = evs.filter((e) => !hide[e.kind]);
    const first = new Date(y, m - 1, 1).getDay(), dim = new Date(y, m, 0).getDate(), prevDim = new Date(y, m - 1, 0).getDate();
    const cells = [];
    for (let i = 0; i < first; i++) cells.push({ d: prevDim - first + 1 + i, other: true });
    for (let d = 1; d <= dim; d++) cells.push({ d, other: false });
    while (cells.length % 7) cells.push({ d: cells.length - first - dim + 1, other: true });
    const weeks = [];
    cells.forEach((c, i) => {
      if (i % 7 === 0) weeks.push([]);
      const dow = i % 7;
      if (c.other) { weeks[weeks.length - 1].push(`<div class="cell other" aria-hidden="true"><div class="dn"><span class="n">${c.d}</span></div></div>`); return; }
      const date = ymd(y, m, c.d);
      const list = vis.filter((e) => e.date === date);
      const isToday = y === t.getFullYear() && m === t.getMonth() + 1 && c.d === t.getDate();
      const season = seasonOf(new Date(y, m - 1, c.d));
      const color = dow === 0 ? 'var(--sun)' : dow === 6 ? 'var(--sat)' : 'var(--text)';
      const chips = list.slice(0, 3).map((e) => {
        const [dot, bd, op] = evStyle(e);
        return `<span class="ev ${e.virtual ? 'plan' : ''}" style="background:${hexA(KC[e.kind], isDark() ? 0.16 : 0.09)};opacity:${op}"><i class="dot" style="width:7px;height:7px;background:${dot};border:${bd};box-sizing:border-box"></i><span class="t"><b style="color:${KC[e.kind]}">${esc(kindLabel(e))}</b> ${esc(e.ref || e.series || '본문 미정')}</span></span>`;
      }).join('');
      weeks[weeks.length - 1].push(`<button type="button" class="cell ${c.d === selD ? 'sel' : ''}" data-d="${c.d}" aria-label="${m}월 ${c.d}일 설교 ${list.length}개">
        <span class="dn"><span class="n ${isToday ? 'today' : ''}" style="${isToday ? '' : 'color:' + color}">${c.d}</span>${season ? `<span class="sn">${esc(season)}</span>` : ''}</span>
        ${chips}${list.length > 3 ? `<span class="more">+${list.length - 3}개 더</span>` : ''}</button>`);
    });

    el.innerHTML = `
      <div class="head">
        <h1>설교 달력</h1>
        <div class="row" style="gap:4px;margin-left:8px">
          <button type="button" class="icon-btn" data-act="prev" aria-label="이전 달">${svgIcon(IC.left)}</button>
          <span class="num" style="min-width:128px;text-align:center;font-size:19px;font-weight:800">${y}년 ${m}월</span>
          <button type="button" class="icon-btn" data-act="next" aria-label="다음 달">${svgIcon(IC.right)}</button>
          <button type="button" class="btn sm" data-act="today" style="margin-left:4px">오늘</button>
        </div>
        <span class="grow"></span>
        <a class="btn brand" href="#sermon?new=1&date=${ymd(y, m, selD)}">${svgIcon(IC.plus)}설교 일정 더하기</a>
      </div>
      <div class="row wrap" style="gap:6px">
        <span class="muted" style="font-size:12px;font-weight:700;margin-right:4px">보기</span>
        ${KINDS.map((k) => `<button type="button" class="pill sm" data-kind="${k}" aria-pressed="${!hide[k]}" style="border-radius:15px;height:30px;font-size:12px;${hide[k] ? 'color:var(--faint);background:transparent' : 'background:var(--panel)'}"><span class="dot" style="background:${KC[k]};margin-right:6px"></span>${k}</button>`).join('')}
        <span class="grow"></span>
        <span class="row wrap" style="gap:12px;font-size:12px;color:var(--muted)">
          <span class="st">${statusDot('본문 미정')}본문 미정</span><span class="st">${statusDot('준비 중')}준비 중</span>
          <span class="st">${statusDot('원고 완료')}원고 완료</span><span class="st">${statusDot('설교함')}설교함</span>
          <span class="st"><i style="border:1px dashed var(--line2);border-radius:3px;width:14px"></i>반복 일정</span></span>
      </div>
      <div class="split">
        <section class="main" aria-label="${y}년 ${m}월 달력" style="overflow-x:auto">
          <div class="cal"><div class="wk">${DOW.map((d, i) => `<div class="dow" style="color:${i === 0 ? 'var(--sun)' : i === 6 ? 'var(--sat)' : 'var(--muted)'}">${d}</div>`).join('')}</div>
          ${weeks.map((w) => `<div class="wk">${w.join('')}</div>`).join('')}</div>
        </section>
        <aside class="aside" id="calside"></aside>
      </div>${footer()}`;
    renderSide(evs);
    el.onclick = (e) => {
      const c = e.target.closest('[data-d]');
      if (c) { selD = +c.dataset.d; $$('.cell', el).forEach((x) => x.classList.toggle('sel', x === c)); renderSide(monthEvents()); return; }
      const k = e.target.closest('[data-kind]');
      if (k) { hide[k.dataset.kind] = !hide[k.dataset.kind]; render(el, {}); return; }
      const a = e.target.closest('[data-act]');
      if (!a) return;
      const act = a.dataset.act;
      if (act === 'prev' || act === 'next') { m += act === 'prev' ? -1 : 1; if (m < 1) { m = 12; y--; } if (m > 12) { m = 1; y++; } selD = 1; render(el, {}); }
      if (act === 'today') { const t2 = today(); y = t2.getFullYear(); m = t2.getMonth() + 1; selD = t2.getDate(); render(el, {}); }
      if (act === 'rule-add') editRule(null);
      if (act === 'rule-edit') editRule(S.data.rules.find((r) => r.id === a.dataset.id));
      if (act === 'make') { const ev = monthEvents().filter((x) => x.virtual)[+a.dataset.i]; if (ev) go('sermon', newSermonQuery(ev)); }
    };
  }

  function renderSide(evs) {
    const side = $('#calside');
    if (!side) return;
    const date = ymd(y, m, selD);
    const d = new Date(y, m - 1, selD);
    const items = evs.filter((e) => e.date === date);
    const virt = evs.filter((e) => e.virtual);
    const season = seasonOf(d);
    const cnt = (st) => evs.filter((e) => e.status === st).length;
    const all = evs.length || 1;
    const sum = { done: cnt('설교함'), ready: cnt('원고 완료'), prep: cnt('준비 중'), plan: cnt('본문 미정') };
    // 알림: 3일 안에 원고가 준비되지 않은 설교
    const t = today(), soon = addDays(t, 3);
    const alerts = upcomingEvents(t, 3).filter((e) => e.status === '본문 미정' || e.status === '준비 중');
    side.innerHTML = `
      <section class="card" aria-label="고른 날">
        <div><div style="font-size:12px;font-weight:800;color:var(--brand)">${esc(season)}</div><h2 style="font-size:20px;margin-top:2px">${m}월 ${selD}일 ${dayName(d)}</h2></div>
        ${items.length ? items.map((e) => {
          const p = (e.passages || [])[0];
          const vi = virt.indexOf(e);
          return `<div class="tile col" style="gap:6px;padding:12px">
            <div class="row wrap" style="gap:6px;font-size:12px"><b class="num">${esc(e.time || '')}</b>${kindChip(e.kind, kindLabel(e))}${e.virtual ? '<span class="chip sm" style="background:transparent;border:1px dashed var(--line2)">반복 일정</span>' : ''}<span class="grow"></span>${statusHtml(e.status)}</div>
            <div style="font-size:15px;font-weight:800;line-height:1.4;${e.title ? '' : 'color:var(--faint)'}">${esc(e.title || '제목 미정')}</div>
            <div class="row wrap" style="gap:4px 10px;font-size:12px;color:var(--muted)"><b style="color:var(--text)">${esc(e.ref || '본문 미정')}</b><span>${esc(e.place || '')}</span>${e.series ? `<span>${esc(e.series)}${e.seriesNo ? ' ' + e.seriesNo : ''}</span>` : ''}</div>
            <div class="row wrap" style="gap:6px;padding-top:2px">
              ${e.virtual ? `<button type="button" class="btn xs brand" data-act="make" data-i="${vi}">이 일정으로 기록 만들기</button>`
                : `<a class="btn xs" href="#sermon?id=${e.id}">${e.status === '설교함' ? '설교 기록 보기' : '설교 기록 열기'}</a>
                   ${p && e.status !== '설교함' ? `<a class="btn xs" href="#helper?sid=${e.id}">도우미로 준비</a>` : ''}
                   ${p ? `<a class="btn xs" href="${bibleURL(p)}" target="_blank">HOLYBIBLE ↗</a>` : ''}`}
            </div></div>`;
        }).join('') : '<div class="empty">이날은 설교 일정이 없어요.</div>'}
        <a class="link" href="#sermon?new=1&date=${date}">+ 이날 설교 더하기</a>
      </section>
      <section class="card" aria-label="이번 달 준비">
        <h2>${m}월 준비 현황 <span class="muted" style="font-size:12px;font-weight:600">모두 ${evs.length}편</span></h2>
        <div class="row" style="height:10px;border-radius:5px;overflow:hidden;background:var(--panel2);gap:0">
          <i style="height:100%;width:${sum.done / all * 100}%;background:var(--done)"></i><i style="height:100%;width:${sum.ready / all * 100}%;background:var(--ready)"></i><i style="height:100%;width:${sum.prep / all * 100}%;background:var(--prep)"></i></div>
        <div style="display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:6px;text-align:center">
          ${[['설교함', sum.done], ['원고 완료', sum.ready], ['준비 중', sum.prep], ['본문 미정', sum.plan]].map(([l, n]) => `<div><div style="font-size:20px;font-weight:800">${n}</div><div class="muted" style="font-size:11px">${l}</div></div>`).join('')}
        </div>
        ${alerts.length ? `<div class="note"><b>알림</b> · ${alerts.slice(0, 3).map((e) => `${fmtShort(e.date)} ${esc(kindLabel(e))} — ${e.status === '본문 미정' ? '본문이 아직 정해지지 않았어요' : '원고가 아직 준비 중이에요'}`).join('<br>')}</div>` : '<div class="note" style="background:var(--ok-bg)"><b style="color:var(--ok)">좋아요</b> · 사흘 안에 준비가 늦은 설교가 없어요.</div>'}
      </section>
      <section class="card" aria-label="반복 일정">
        <div class="ch"><h2>반복 일정</h2><button type="button" class="link" data-act="rule-add">+ 더하기</button></div>
        ${S.data.rules.length ? S.data.rules.map((r) => {
          const ser = S.data.seriesById[r.seriesId];
          return `<button type="button" class="tile row" data-act="rule-edit" data-id="${r.id}" style="border:0;text-align:left;font-size:13px;${r.active === false ? 'opacity:.5' : ''}">
            <span class="dot" style="background:${KC[r.kind]}"></span><span class="grow ellipsis"><b>${r.every === 2 ? '격주' : '매주'} ${DOW[r.dow]}요일${r.dow === 0 ? '(주일)' : ''} ${esc(r.time || '')}</b> · ${esc(r.kind)}${ser ? ' · ' + esc(ser.name) + ' 이어서' : ''}${r.place ? ' · ' + esc(r.place) : ''}</span></button>`;
        }).join('') : '<div class="empty">매주 하는 설교(주일 오전, 수요 등)를 정해 두면 달력에 예정이 저절로 채워져요.</div>'}
        <p class="faint" style="margin:0;font-size:12px;line-height:1.6">강해 시리즈와 이어 두면, 시리즈의 ‘다음 본문’을 가장 가까운 예정 칸에 채워 둬요.</p>
      </section>`;
  }

  async function editRule(r) {
    const x = r ? { ...r } : { dow: 0, every: 1, time: '11:00', kind: '주일', place: (S.settings.ui && S.settings.ui.homePlace) || '본 교회', seriesId: '', note: '', active: true, start: iso(today()) };
    const sopts = S.data.series.filter((s) => !s.done || s.id === x.seriesId).map((s) => `<option value="${s.id}" ${s.id === x.seriesId ? 'selected' : ''}>${esc(s.name)}</option>`).join('');
    const v = await modal({
      title: r ? '반복 일정 고치기' : '반복 일정 더하기',
      body: `<div style="display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px">
        <label class="field">요일<select class="select" id="r-dow">${DOW.map((d, i) => `<option value="${i}" ${i === x.dow ? 'selected' : ''}>${i === 0 ? '주일' : d + '요일'}</option>`).join('')}</select></label>
        <label class="field">얼마나 자주<select class="select" id="r-every"><option value="1">매주</option><option value="2" ${x.every === 2 ? 'selected' : ''}>격주</option></select></label>
        <label class="field">시간<input class="input" id="r-time" value="${esc(x.time)}" placeholder="11:00"></label>
        <label class="field">종류<select class="select" id="r-kind">${KINDS.map((k) => `<option ${k === x.kind ? 'selected' : ''}>${k}</option>`).join('')}</select></label>
        <label class="field">장소<input class="input" id="r-place" value="${esc(x.place)}"></label>
        <label class="field">강해 시리즈 (이어서 채우기)<select class="select" id="r-series"><option value="">없음</option>${sopts}</select></label>
        <label class="field" style="grid-column:1/-1">메모 (시리즈가 없을 때 달력에 보이는 이름)<input class="input" id="r-note" value="${esc(x.note || '')}" placeholder="예: 금요기도회"></label>
      </div><label class="check"><input type="checkbox" id="r-active" ${x.active !== false ? 'checked' : ''}>이 반복 일정 쓰기</label>`,
      buttons: [...(r ? [{ label: '지우기', kind: 'danger ghost', value: 'delete' }] : []), { label: '취소', value: null }, {
        label: '저장', kind: 'brand', value: (el) => ({ ...x, dow: +$('#r-dow', el).value, every: +$('#r-every', el).value, time: $('#r-time', el).value.trim(), kind: $('#r-kind', el).value,
          place: $('#r-place', el).value.trim(), seriesId: $('#r-series', el).value, note: $('#r-note', el).value.trim(), active: $('#r-active', el).checked, start: x.start || iso(today()) }),
      }],
    });
    try {
      if (v === 'delete') { if (await confirmBox('반복 일정을 지울까요?', '이미 만든 설교 기록은 그대로 남아요.', '지우기', true)) { await api('/api/rule/save', { delete: r.id }); await loadData(); refresh(); toast('반복 일정을 지웠어요'); } return; }
      if (!v) return;
      await api('/api/rule/save', { rule: v }); await loadData(); refresh(); toast('반복 일정을 저장했어요');
    } catch (e) { fail(e); }
  }

  return { render: (el, q) => render(el, q || {}), refresh: () => { const el = $('#main .page'); if (el) render(el, {}); } };
})();
