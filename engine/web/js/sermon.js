/* 설교 기록하기 — 본문 · 제목 · 날짜 · 장소 · 종류 · 절기 · 내용 · 시리즈 · 요리문답 · 첨부 · 예화 */
'use strict';

Screens.sermon = (() => {
  let F = null;       // 지금 고치는 설교
  let dirty = false;
  let seasonTouched = false;
  let el = null;

  function blank(q) {
    const t = today();
    const date = q.date || iso(t);
    const d = parseISO(date);
    const dowKind = { 0: '주일', 3: '수요', 5: '금요' }[d.getDay()] || '새벽';
    const places = placeStats();
    const home = (S.settings.ui && S.settings.ui.homePlace) || (places[0] && places[0][0]) || '본 교회';
    const conf = [];
    if (q.conf) { const [doc, n] = q.conf.split(':'); conf.push({ doc, n: +n }); }
    const past = d < t;
    return {
      id: '', date, time: q.time || (d.getDay() === 0 ? '11:00' : ''), kind: q.kind || dowKind, kindOther: '', season: '',
      title: '', passageText: q.ref || '', passages: [], bibleText: '', thesis: '', points: [''], summary: '', place: q.place || home,
      status: past ? '설교함' : (q.ref ? '준비 중' : '본문 미정'), seriesId: q.series || '', seriesNo: 0, confessions: conf, illus: [], refs: '',
      files: [], helper: [], meditation: '',
    };
  }

  function placeStats() {
    const c = {};
    for (const s of S.data.sermons) if (s.place) c[s.place] = (c[s.place] || 0) + 1;
    return Object.entries(c).sort((a, b) => b[1] - a[1]);
  }

  async function render(page, q, changed) {
    el = page;
    if (changed || !F) {
      dirty = false; seasonTouched = false;
      if (q.id) {
        const s = await api('/api/sermon?id=' + encodeURIComponent(q.id));
        F = JSON.parse(JSON.stringify(s));
        F.helper = Object.keys(s.helper || {});
        if (!F.points || !F.points.length) F.points = [''];
        seasonTouched = true;
      } else {
        F = blank(q);
        if (F.seriesId) suggestSeriesNo();
        autoSeason();
      }
    }
    draw();
  }

  function autoSeason() {
    if (seasonTouched) return;
    const opt = seasonOption(seasonOf(parseISO(F.date)));
    F.season = opt || '';
  }
  function suggestSeriesNo() {
    if (!F.seriesId) { F.seriesNo = 0; return; }
    const n = S.data.sermons.filter((s) => s.seriesId === F.seriesId && s.id !== F.id && s.date < F.date).length;
    F.seriesNo = n + 1;
  }

  function draw() {
    const isNew = !F.id;
    const places = placeStats().slice(0, 6).map(([p]) => p).filter((p) => p !== F.place);
    const seasonIsCustom = F.season && !SEASON_OPTIONS.includes(F.season);
    el.innerHTML = `
      <div class="head">
        <div class="t"><div class="crumb"><a href="#records">설교 기록</a> › ${isNew ? '새 설교' : esc(F.title || '제목 미정')}</div>
          <h1 style="margin-top:4px">${isNew ? '설교 기록하기' : '설교 기록'}</h1></div>
        <button type="button" class="btn" data-act="helper">${svgIcon(IC.bulb)}설교 도우미로 준비</button>
        ${isNew ? '' : `<button type="button" class="btn ghost danger" data-act="delete">${svgIcon(IC.trash)}지우기</button>`}
        <button type="button" class="btn ghost" data-act="cancel">${isNew ? '취소' : '닫기'}</button>
        <button type="button" class="btn brand" data-act="save">저장 <span style="opacity:.7;font-weight:700">⌘S</span></button>
      </div>
      <div id="dups"></div>
      <div class="split">
        <div class="main">
          <section class="card" aria-label="본문">
            <h2>본문</h2>
            <div class="row wrap" style="gap:10px;align-items:flex-end">
              <label class="field" style="flex:1 1 260px">성경 구절 <span class="faint" style="font-weight:600">· 여러 곳은 ; 로 (예: 롬 8:28-30; 엡 1:4)</span>
                <input class="input" id="f-ref" value="${esc(F.passageText)}" placeholder="예: 로마서 8:28-30, 롬 8장 28~30절" style="font-size:16px;font-weight:700"></label>
            </div>
            <div class="row wrap" id="refchips" style="gap:6px;min-height:20px"></div>
            <label class="field">본문 직접 넣기 <span class="faint" style="font-weight:600">· 쓰시는 번역본의 본문을 붙여 넣으세요</span>
              <textarea class="textarea serif" id="f-bible" rows="5" placeholder="예) 28 우리가 알거니와 하나님을 사랑하는 자 곧 그의 뜻대로 부르심을 입은 자들에게는…">${esc(F.bibleText)}</textarea></label>
          </section>

          <section class="card" aria-label="제목 날짜 장소">
            <label class="field">설교 제목<input class="input big" id="f-title" value="${esc(F.title)}" placeholder="제목이 아직 없으면 비워 두어도 돼요"></label>
            <div class="row wrap" style="gap:12px;align-items:flex-start">
              <label class="field" style="flex:1 1 170px">날짜<input class="input" type="date" id="f-date" value="${esc(F.date)}"></label>
              <label class="field" style="flex:0 0 130px">시간<input class="input" type="time" id="f-time" value="${esc(F.time)}"></label>
              <div style="flex:2 1 260px;min-width:0">
                <label class="field">장소<input class="input" id="f-place" value="${esc(F.place)}" list="placelist"></label>
                <datalist id="placelist">${placeStats().map(([p]) => `<option value="${esc(p)}">`).join('')}</datalist>
                ${places.length ? `<div class="row wrap" style="gap:5px;font-size:12px;margin-top:6px"><span class="faint">자주 쓰는 곳</span>${places.map((p) => `<button type="button" class="chip sm" data-place="${esc(p)}" style="background:transparent;border:1px solid var(--line);font-weight:600;color:var(--muted)">${esc(p)}</button>`).join('')}</div>` : ''}
              </div>
            </div>
          </section>

          <section class="card" aria-label="종류와 절기">
            <div class="col" style="gap:8px"><h2>설교 종류</h2>
              <div class="row wrap">${KINDS.map((k) => `<button type="button" class="pill ${F.kind === k ? 'on' : ''}" data-kind="${k}" aria-pressed="${F.kind === k}">${k}</button>`).join('')}</div>
              ${F.kind === '그 외' ? `<input class="input" id="f-kindother" value="${esc(F.kindOther)}" placeholder="예: 장례 예배, 결혼 예배, 임직식, 부흥회, 심방" style="max-width:360px;height:38px;font-size:14px">` : ''}
            </div>
            <div class="col" style="gap:8px"><h2>절기 <span class="faint" style="font-size:12px;font-weight:600">날짜를 보고 알맞은 절기를 먼저 골라 둬요${seasonOf(parseISO(F.date)) ? ' · 이날은 ' + esc(seasonOf(parseISO(F.date))) : ''}</span></h2>
              <div class="row wrap" style="gap:6px">${SEASON_OPTIONS.map((s) => {
                const on = s === '없음' ? !F.season : s === '직접 입력' ? seasonIsCustom : F.season === s;
                return `<button type="button" class="pill sm ${on ? 'on' : ''}" data-season="${s}" aria-pressed="${on}">${s}</button>`;
              }).join('')}</div>
              ${seasonIsCustom || F._customSeason ? `<input class="input" id="f-season" value="${esc(seasonIsCustom ? F.season : '')}" placeholder="절기 이름" style="max-width:300px;height:38px;font-size:14px">` : ''}
            </div>
          </section>

          <section class="card" aria-label="설교 내용">
            <h2>설교 내용</h2>
            <label class="field">중심 명제 (한 문장)<input class="input" id="f-thesis" value="${esc(F.thesis)}" placeholder="이 설교가 전하려는 한 문장"></label>
            <div class="col" style="gap:6px"><div class="row" style="justify-content:space-between"><span class="muted" style="font-size:12px;font-weight:700">대지</span><button type="button" class="link" data-act="addpoint">+ 대지 더하기</button></div>
              ${F.points.map((p, i) => `<div class="row" style="gap:8px"><span style="color:var(--brand);font-weight:800;width:18px;text-align:center">${i + 1}</span>
                <input class="input" data-point="${i}" value="${esc(p)}" style="height:38px;font-size:14px" placeholder="대지 ${i + 1}">
                <button type="button" class="icon-btn" data-delpoint="${i}" aria-label="대지 ${i + 1} 빼기" style="width:32px;height:32px">${svgIcon(IC.x, 14)}</button></div>`).join('')}
            </div>
            <label class="field">요약 · 메모<textarea class="textarea" id="f-summary" rows="4">${esc(F.summary)}</textarea></label>
            <label class="field">묵상 기록<textarea class="textarea serif" id="f-med" rows="3" style="font-size:15px">${esc(F.meditation)}</textarea></label>
          </section>
          ${F.helper && F.helper.length ? `<section class="card tight" aria-label="설교 도우미 결과"><div class="ch"><h2>설교 도우미로 정리한 것</h2><a href="#helper?sid=${F.id}">도우미 열기</a></div>
            <div class="row wrap" style="gap:6px">${F.helper.map((k) => `<a class="chip" href="#helper?sid=${F.id}&tab=${k}">${esc({ sum: '정리 · 요약', str: '구조', med: '묵상', outline: '설교 개요', ref: '개혁주의 관점' }[k] || k)}</a>`).join('')}</div></section>` : ''}
        </div>

        <aside class="aside">
          <section class="card tight" aria-label="준비 상태"><h2>준비 상태</h2>
            <div class="seg">${STATUSES.map((s) => `<button type="button" data-status="${s}" class="${F.status === s ? 'on' : ''}" aria-pressed="${F.status === s}">${s}</button>`).join('')}</div></section>
          <section class="card tight" aria-label="강해 시리즈" id="seriesbox"></section>
          <section class="card tight" aria-label="요리문답 신앙고백">
            <div class="ch"><h2>요리문답 · 신앙고백</h2><button type="button" class="link" data-act="conf">+ 연결하기</button></div>
            <div class="row wrap" style="gap:6px">${F.confessions.length ? F.confessions.map((c, i) => `<span class="chip">${esc(confLabel(c))}<button type="button" class="x" data-delconf="${i}" aria-label="빼기">×</button></span>`).join('') : '<span class="faint" style="font-size:13px">연결한 조항이 없어요</span>'}</div>
          </section>
          <section class="card tight" aria-label="자료 첨부" id="filebox"></section>
          <section class="card tight" aria-label="쓴 예화" id="illusbox"></section>
          <section class="card tight" aria-label="참고한 자료"><h2>참고한 자료</h2>
            <textarea class="textarea" id="f-refs" rows="3" placeholder="예) 칼빈 로마서 주석 312–318쪽" style="min-height:70px">${esc(F.refs)}</textarea></section>
        </aside>
      </div>${footer()}`;
    bind();
    updateRefs();
    drawSeries();
    drawFiles();
    drawIllus();
  }

  const mark = () => { dirty = true; };
  function bind() {
    const on = (id, fn, ev = 'input') => { const x = $('#' + id, el); if (x) x.addEventListener(ev, (e) => { fn(e.target.value); mark(); }); };
    on('f-ref', (v) => { F.passageText = v; updateRefsSoon(); });
    on('f-bible', (v) => { F.bibleText = v; });
    on('f-title', (v) => { F.title = v; });
    on('f-date', (v) => { if (!v) return; F.date = v; autoSeason(); if (F.seriesId && !F.id) suggestSeriesNo(); keepScroll(draw); }, 'change');
    on('f-time', (v) => { F.time = v; });
    on('f-place', (v) => { F.place = v; updateDups(); drawIllus(); });
    on('f-kindother', (v) => { F.kindOther = v; });
    on('f-season', (v) => { F.season = v.trim(); seasonTouched = true; });
    on('f-thesis', (v) => { F.thesis = v; });
    on('f-summary', (v) => { F.summary = v; });
    on('f-med', (v) => { F.meditation = v; });
    on('f-refs', (v) => { F.refs = v; });
    $$('[data-point]', el).forEach((x) => x.addEventListener('input', (e) => { F.points[+x.dataset.point] = e.target.value; mark(); }));
    el.onclick = async (e) => {
      const t = e.target.closest('button, a');
      if (!t) return;
      const d = t.dataset;
      if (d.kind) { F.kind = d.kind; mark(); return keepScroll(draw); }
      if (d.season) {
        seasonTouched = true; mark();
        if (d.season === '없음') { F.season = ''; F._customSeason = false; } else if (d.season === '직접 입력') { F._customSeason = true; if (SEASON_OPTIONS.includes(F.season)) F.season = ''; } else { F.season = d.season; F._customSeason = false; }
        return keepScroll(draw);
      }
      if (d.status) { F.status = d.status; mark(); $$('[data-status]', el).forEach((b) => { b.classList.toggle('on', b === t); b.setAttribute('aria-pressed', b === t); }); return; }
      if (d.place) { F.place = d.place; $('#f-place', el).value = d.place; mark(); updateDups(); drawIllus(); return; }
      if (d.delpoint !== undefined) { F.points.splice(+d.delpoint, 1); if (!F.points.length) F.points = ['']; mark(); return keepScroll(draw); }
      if (d.delconf !== undefined) { F.confessions.splice(+d.delconf, 1); mark(); return keepScroll(draw); }
      switch (d.act) {
        case 'addpoint': F.points.push(''); mark(); keepScroll(draw); setTimeout(() => { const xs = $$('[data-point]', el); xs[xs.length - 1].focus(); }, 0); break;
        case 'save': save(); break;
        case 'cancel': history.length > 1 ? history.back() : go('records'); break;
        case 'delete': del(); break;
        case 'helper': toHelper(); break;
        case 'conf': pickConf(); break;
      }
    };
  }
  function keepScroll(fn) { const m = $('#main'); const top = m.scrollTop; fn(); m.scrollTop = top; }

  // 본문 알아보기 (엔진이 읽어요)
  const updateRefsSoon = debounce(updateRefs, 280);
  async function updateRefs() {
    const box = $('#refchips', el);
    if (!box) return;
    if (!F.passageText.trim()) { F.passages = []; box.innerHTML = '<span class="faint" style="font-size:12px">본문을 적으면 알아본 결과가 여기에 보여요</span>'; updateDups(); return; }
    try {
      const r = await api('/api/parse', { text: F.passageText });
      F.passages = r.passages;
      box.innerHTML = r.passages.map((p) => {
        if (!(p.book > 0)) return `<span class="chip" style="background:var(--warn-bg);color:var(--warn)">알아보지 못했어요: ${esc(p.raw || p.ref)}</span>`;
        const b = bookOf(p.book);
        return `<span class="chip"><span class="sq" style="background:${bookColor(b)}"></span>${esc(p.ref)} · ${b.ot ? '구약' : '신약'} · ${esc(b.group)}</span>
          <a class="btn xs ink" href="${bibleURL(p)}" target="_blank">HOLYBIBLE에서 열기 ${svgIcon(IC.ext, 12)}</a>`;
      }).join('');
    } catch (e) { /* 다음에 다시 */ }
    updateDups();
  }

  function updateDups() {
    const box = $('#dups', el);
    if (!box) return;
    const ps = (F.passages || []).filter((p) => p.book > 0);
    if (!ps.length) { box.innerHTML = ''; return; }
    const list = sermonsOverlapping(ps, F.id).slice().reverse();
    if (!list.length) { box.innerHTML = ''; return; }
    const place = (F.place || '').trim();
    const same = list.filter((s) => place && s.place === place);
    const other = list.filter((s) => !(place && s.place === place));
    const line = (s) => `${fmtDate(s.date)} ${esc(kindLabel(s))} · ‘${esc(s.title || '제목 미정')}’ · ${esc(s._ref)}`;
    box.innerHTML = `<div role="status" class="col" style="gap:6px;padding:14px 16px;border-radius:14px;background:var(--warn-bg);border:1px solid var(--warn-line)">
      <div class="row wrap" style="gap:8px;font-size:14px">${svgIcon(IC.warn, 18).replace('stroke="currentColor"', 'stroke="var(--warn)"')}
        <b>${same.length ? esc(place) + '에서 같은 본문으로 설교한 적이 있어요.' : '다른 곳에서 같은 본문으로 설교한 적이 있어요.'}</b>
        <span class="muted">${line(same[0] || other[0])}</span>
        <a class="link" href="#sermon?id=${(same[0] || other[0]).id}" style="margin-left:auto;color:var(--warn)">그 설교 보기</a></div>
      ${list.length > 1 ? `<div class="muted" style="font-size:13px;padding-left:26px;line-height:1.7">${list.slice(1, 4).map((s) => `${line(s)}${s.place ? ' · ' + esc(s.place) : ''}`).join('<br>')}${list.length > 4 ? '<br>외 ' + (list.length - 4) + '편' : ''}</div>` : ''}
    </div>`;
  }

  function drawSeries() {
    const box = $('#seriesbox', el);
    const ser = S.data.seriesById[F.seriesId];
    const opts = S.data.series.filter((s) => !s.done || s.id === F.seriesId);
    let prog = '';
    if (ser) {
      const done = S.data.sermons.filter((s) => s.seriesId === ser.id).length;
      const pct = ser.total ? Math.min(100, Math.round(done / ser.total * 100)) : 0;
      prog = `<div style="font-size:15px;font-weight:800">${esc(ser.name)} <span class="muted" style="font-weight:600">· ${F.seriesNo || '?'}번째${ser.total ? ' / ' + ser.total : ''}</span></div>
        ${ser.total ? `<div class="progress"><i style="width:${pct}%;background:${ser.color || 'var(--brand)'}"></i></div>` : ''}
        ${ser.nextRef ? `<div class="muted" style="font-size:13px">시리즈의 다음 본문 <b style="color:var(--text)">${esc(ser.nextRef)}</b></div>` : ''}`;
    }
    box.innerHTML = `<div class="ch"><h2>강해 시리즈</h2><a href="#series">시리즈 관리</a></div>
      <div class="row" style="gap:8px"><select class="select grow" id="f-series" style="height:38px;font-size:14px"><option value="">시리즈 아님</option>${opts.map((s) => `<option value="${s.id}" ${s.id === F.seriesId ? 'selected' : ''}>${esc(s.name)}</option>`).join('')}</select>
        ${F.seriesId ? `<label class="row" style="gap:4px;font-size:13px;color:var(--muted)"><input class="input num" id="f-serno" type="number" min="1" value="${F.seriesNo || ''}" style="width:64px;height:38px;font-size:14px;padding:0 8px">번째</label>` : ''}</div>${prog}`;
    $('#f-series', box).onchange = (e) => { F.seriesId = e.target.value; suggestSeriesNo(); mark(); drawSeries(); };
    const no = $('#f-serno', box);
    if (no) no.oninput = (e) => { F.seriesNo = +e.target.value || 0; mark(); };
  }

  const FT = { pdf: ['PDF', '#8B5A2B'], hwp: ['HWP', '#2E7DD7'], hwpx: ['HWP', '#2E7DD7'], doc: ['DOC', '#2B579A'], docx: ['DOC', '#2B579A'], pages: ['PAGES', '#F29D38'],
    m4a: ['M4A', '#C2533A'], mp3: ['MP3', '#C2533A'], wav: ['WAV', '#C2533A'], mp4: ['MP4', '#7B3FB8'], mov: ['MOV', '#7B3FB8'], key: ['KEY', '#1F7AE0'], pptx: ['PPT', '#C4462A'],
    jpg: ['JPG', '#2F8F6B'], jpeg: ['JPG', '#2F8F6B'], png: ['PNG', '#2F8F6B'], txt: ['TXT', '#6B7083'] };
  function drawFiles() {
    const box = $('#filebox', el);
    const files = F.files || [];
    const size = (n) => n > 1 << 20 ? (n / (1 << 20)).toFixed(1) + 'MB' : Math.max(1, Math.round(n / 1024)) + 'KB';
    box.innerHTML = `<h2>자료 첨부</h2>
      ${files.map((f) => {
        const ext = (f.name.split('.').pop() || '').toLowerCase();
        const [lab, col] = FT[ext] || [ext.slice(0, 4).toUpperCase() || 'FILE', '#6B7083'];
        return `<div class="tile row" style="gap:10px;padding:9px 10px"><span class="ftype" style="background:${col}">${esc(lab)}</span>
          <button type="button" class="grow ellipsis" data-file="${f.id}" data-fa="open" style="border:0;background:none;padding:0;text-align:left;font-size:13px;font-weight:700" title="열기">${esc(f.name)}</button>
          <span class="faint" style="font-size:12px">${size(f.size)}</span>
          <button type="button" class="icon-btn" data-file="${f.id}" data-fa="reveal" title="Finder에서 보기" style="width:28px;height:28px">${svgIcon(IC.folder, 13)}</button>
          <button type="button" class="icon-btn" data-file="${f.id}" data-fa="remove" title="빼기" style="width:28px;height:28px">${svgIcon(IC.x, 13)}</button></div>`;
      }).join('')}
      <label class="drop" id="drop">원고 · 녹음 · 영상 · 주보를 여기에 끌어 놓거나 <b style="color:var(--brand)">눌러서 고르세요</b><input type="file" id="f-files" multiple hidden></label>`;
    const drop = $('#drop', box);
    drop.addEventListener('dragover', (e) => { e.preventDefault(); drop.classList.add('over'); });
    drop.addEventListener('dragleave', () => drop.classList.remove('over'));
    drop.addEventListener('drop', (e) => { e.preventDefault(); drop.classList.remove('over'); upload(e.dataTransfer.files); });
    $('#f-files', box).onchange = (e) => upload(e.target.files);
    box.onclick = async (e) => {
      const b = e.target.closest('[data-fa]');
      if (!b) return;
      if (b.dataset.fa === 'remove' && !(await confirmBox('첨부 파일을 뺄까요?', '앱 안에 보관한 이 파일을 지워요. 원래 파일은 그대로예요.', '빼기', true))) return;
      try {
        const r = await api('/api/file/action', { sid: F.id, fid: b.dataset.file, action: b.dataset.fa });
        F.files = r.files; if (b.dataset.fa === 'remove') { drawFiles(); await loadData(); }
      } catch (err) { fail(err); }
    };
  }
  async function upload(files) {
    if (!files || !files.length) return;
    if (!F.id) { const ok = await save(true); if (!ok) return; }
    const fd = new FormData();
    fd.append('sid', F.id);
    for (const f of files) fd.append('file', f, f.name);
    toast(files.length + '개 파일을 넣는 중…');
    try { const r = await api('/api/file/add', fd); F.files = r.files; drawFiles(); await loadData(); toast('파일을 넣었어요'); } catch (e) { fail(e); }
  }

  function illusUse(id) {
    const used = S.data.sermons.filter((s) => s.id !== F.id && (s.illus || []).includes(id)).sort((a, b) => b.date.localeCompare(a.date));
    const place = (F.place || '').trim();
    const same = used.find((s) => place && s.place === place);
    if (same) return `<div class="warn-text">${esc(place)}에서 ${fmtDate(same.date)}에 이미 썼어요</div>`;
    if (used.length) return `<div class="muted" style="font-size:12px">${esc(used[0].place || '다른 곳')}에서 ${fmtDate(used[0].date)}에 썼어요 · 이곳에서는 처음</div>`;
    return `<div class="ok-text">${place ? esc(place) + '에서는 ' : ''}처음 써요</div>`;
  }
  function drawIllus() {
    const box = $('#illusbox', el);
    if (!box) return;
    box.innerHTML = `<div class="ch"><h2>쓴 예화</h2><button type="button" class="link" id="pickillus">보관함에서 고르기</button></div>
      ${F.illus.length ? F.illus.map((id) => { const x = S.data.illusById[id]; if (!x) return ''; return `<div class="tile col" style="gap:3px">
        <div class="row"><b class="grow" style="font-size:14px">${esc(x.title)}</b><button type="button" class="chip sm" data-delillus="${id}" style="background:transparent">빼기</button></div>${illusUse(id)}</div>`; }).join('')
        : '<span class="faint" style="font-size:13px">이 설교에 쓴 예화를 골라 두면, 같은 회중에게 또 쓰지 않도록 알려 드려요</span>'}`;
    $('#pickillus', box).onclick = pickIllus;
    box.querySelectorAll('[data-delillus]').forEach((b) => b.onclick = () => { F.illus = F.illus.filter((x) => x !== b.dataset.delillus); mark(); drawIllus(); });
  }
  async function pickIllus() {
    const sel = new Set(F.illus);
    const listHtml = (q) => S.data.illus.filter((x) => !q || (x.title + ' ' + x.body + ' ' + (x.tags || []).join(' ')).toLowerCase().includes(q.toLowerCase()))
      .map((x) => `<label class="src ${sel.has(x.id) ? '' : 'off'}" style="cursor:pointer"><input type="checkbox" data-id="${x.id}" ${sel.has(x.id) ? 'checked' : ''}>
        <span class="col grow" style="gap:3px;min-width:0"><b style="font-size:14px">${esc(x.title)}</b><span class="muted ellipsis" style="font-size:12px">${esc((x.body || '').slice(0, 80))}</span>${illusUse(x.id)}</span></label>`).join('')
      || '<div class="empty">보관함에 예화가 없어요. 아래에서 바로 새로 만들 수 있어요.</div>';
    const r = await modal({
      title: '예화 고르기', wide: true,
      body: `<label class="search">${svgIcon(IC.search)}<input id="iq" placeholder="예화 찾기"></label><div id="ilist" class="col" style="gap:8px;max-height:46vh;overflow-y:auto">${listHtml('')}</div>
        <div class="hr"></div><div class="row" style="gap:8px"><input class="input grow" id="inew" placeholder="새 예화 제목 (내용은 예화 보관함에서 채울 수 있어요)" style="height:38px;font-size:14px"><button type="button" class="btn sm" id="iadd">새 예화 만들기</button></div>`,
      buttons: [{ label: '취소', value: null }, { label: '고르기', kind: 'brand', value: () => Array.from(sel) }],
      onOpen: (m) => {
        const bindList = () => $$('#ilist input', m).forEach((c) => c.onchange = () => { c.checked ? sel.add(c.dataset.id) : sel.delete(c.dataset.id); c.closest('.src').classList.toggle('off', !c.checked); });
        bindList();
        $('#iq', m).oninput = (e) => { $('#ilist', m).innerHTML = listHtml(e.target.value.trim()); bindList(); };
        $('#iadd', m).onclick = async () => {
          const t = $('#inew', m).value.trim(); if (!t) return;
          try { const res = await api('/api/illus/save', { illus: { title: t, body: '', source: '', tags: [] } }); await loadData(); sel.add(res.illus.id); $('#inew', m).value = ''; $('#ilist', m).innerHTML = listHtml(''); bindList(); } catch (e) { fail(e); }
        };
      },
    });
    if (r) { F.illus = r; mark(); drawIllus(); }
  }

  async function pickConf() {
    let doc = 'hc';
    const sel = new Set(F.confessions.map((c) => c.doc + ':' + c.n));
    const grid = () => {
      const d = CONFESSIONS[doc];
      return `<div class="row wrap" style="gap:6px">${Object.entries(CONFESSIONS).map(([k, v]) => `<button type="button" class="pill sm ${k === doc ? 'on' : ''}" data-doc="${k}">${v.name}</button>`).join('')}</div>
        <div class="ugrid" style="grid-template-columns:repeat(auto-fill,minmax(${d.min},1fr));max-height:48vh;overflow-y:auto;padding:4px">${Array.from({ length: d.total }, (_, i) => {
          const n = i + 1, on = sel.has(doc + ':' + n);
          return `<button type="button" class="unit ${on ? 'done' : ''}" data-n="${n}" style="min-height:${d.h}"><b>${d.label(n)}</b>${d.theme(n) ? `<span>${esc(d.theme(n))}</span>` : (doc === 'wsc' ? `<span>${esc(wscPart(n))}</span>` : '')}</button>`;
        }).join('')}</div>`;
    };
    const r = await modal({
      title: '요리문답 · 신앙고백 연결하기', wide: true, body: `<div id="cg" class="col">${grid()}</div>`,
      buttons: [{ label: '취소', value: null }, { label: '연결하기', kind: 'brand', value: () => Array.from(sel) }],
      onOpen: (m) => {
        $('#cg', m).onclick = (e) => {
          const dd = e.target.closest('[data-doc]'); if (dd) { doc = dd.dataset.doc; $('#cg', m).innerHTML = grid(); return; }
          const u = e.target.closest('[data-n]'); if (u) { const k = doc + ':' + u.dataset.n; sel.has(k) ? sel.delete(k) : sel.add(k); u.classList.toggle('done'); }
        };
      },
    });
    if (r) {
      const order = Object.keys(CONFESSIONS);
      F.confessions = r.map((k) => { const [d, n] = k.split(':'); return { doc: d, n: +n }; }).sort((a, b) => order.indexOf(a.doc) - order.indexOf(b.doc) || a.n - b.n);
      mark(); keepScroll(draw);
    }
  }

  async function save(quiet) {
    const t = $('#f-title', el); if (t) F.title = t.value;
    if (!F.date) { toast('날짜를 정해 주세요', { err: true }); return false; }
    const body = { ...F, points: F.points.filter((p) => p.trim()) };
    delete body._customSeason;
    try {
      const r = await api('/api/sermon/save', { sermon: body });
      const wasNew = !F.id;
      F.id = r.sermon.id; F.files = r.sermon.files || []; F.passages = r.sermon.passages;
      dirty = false;
      await loadData();
      if (wasNew) { history.replaceState(null, '', '#sermon?id=' + F.id); lastRouteKey = location.hash; }
      if (!quiet) { toast('저장했어요'); keepScroll(draw); }
      return true;
    } catch (e) { fail(e); return false; }
  }

  async function del() {
    if (!(await confirmBox('이 설교 기록을 지울까요?', '지운 기록은 휴지통에 남아서 다시 되살릴 수 있어요 (통계 · 백업 › 휴지통).', '지우기', true))) return;
    try {
      await api('/api/sermon/delete', { id: F.id });
      const id = F.id;
      dirty = false; await loadData();
      go('records');
      toast('지웠어요', { action: '되돌리기', onAction: async () => { try { await api('/api/sermon/restore', { id }); await loadData(); go('sermon', { id }); } catch (e) { fail(e); } } });
    } catch (e) { fail(e); }
  }

  async function toHelper() {
    if (!F.id || dirty) { const ok = await save(true); if (!ok) return; }
    go('helper', { sid: F.id });
  }

  async function canLeave() {
    if (!dirty) return true;
    const r = await modal({ title: '저장하지 않은 내용이 있어요', body: '<p style="margin:0;color:var(--muted)">고친 내용을 저장할까요?</p>',
      buttons: [{ label: '취소', value: 'stay' }, { label: '저장하지 않고 나가기', kind: 'danger ghost', value: 'drop' }, { label: '저장하고 나가기', kind: 'brand', value: 'save' }] });
    if (r === 'save') return await save(true);
    if (r === 'drop') { dirty = false; return true; }
    return false;
  }

  return {
    render, canLeave, save: () => save(),
    refresh: () => { /* 고치는 중에는 다시 그리지 않아요 (다른 곳에서 바뀐 목록만 새로) */ if (!dirty && F && el) { drawIllus(); drawSeries(); } },
    leave: () => { F = null; dirty = false; },
  };
})();
