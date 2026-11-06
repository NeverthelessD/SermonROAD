/* 설교 한꺼번에 등록 — 붙여 넣기 · 엑셀 파일 · 원고 파일 폴더 → 미리 보고 고치기 → 등록 */
'use strict';

Screens.import = (() => {
  let el = null;
  let way = 'paste';
  let text = '';
  let folder = '', recursive = true, attach = true;
  let rows = null;          // 미리 보기
  let onlyIssues = false;
  let busy = false;
  let done = null;          // 등록 결과

  const EXAMPLE = '날짜\t종류\t본문\t제목\t장소\n2019.03.10\t주일\t롬 8:28-30\t하나님의 영원한 작정\t본 교회\n2019.03.13\t수요\t시편 23편\t여호와는 나의 목자시니\t본 교회\n2019.03.17\t주일 오후\t엡 1:3-14\t창세 전에 택하사\t○○교회';

  function render(page) {
    el = page;
    draw();
  }

  const issue = (r) => !r.date ? '날짜' : (r.passageText && !r.refOK) ? '본문' : '';

  function draw() {
    el.innerHTML = `
      <div class="head"><div class="t"><div class="crumb"><a href="#records">설교 기록</a> › 한꺼번에 등록</div>
        <h1 style="margin-top:4px">설교 한꺼번에 등록</h1>
        <p class="sub">앱을 쓰기 전에 했던 설교를 한 번에 넣어요. 먼저 미리 보고 고친 뒤 등록해요. 등록하기 전에 지금 기록을 백업해 둬요.</p></div></div>
      ${done ? doneHtml() : `
      <section class="card" aria-label="넣는 방법">
        <div class="step"><span class="k">1</span><h2>어떻게 넣을까요?</h2></div>
        <div role="radiogroup" style="display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:10px">
          ${[['paste', '표 붙여 넣기', '엑셀 · 한글 · 메모장에서 복사해서 붙여 넣어요. 한 줄에 설교 하나.'],
             ['excel', '엑셀 파일', '엑셀 파일(.xlsx)을 넣어요. 채워 넣을 서식 파일도 드려요.'],
             ['folder', '원고 파일 폴더', '원고 파일 이름에서 날짜 · 본문 · 제목을 읽고, 원고를 설교 기록에 첨부해요.']].map(([k, t, d]) =>
            `<button type="button" role="radio" aria-checked="${way === k}" class="radio-card ${way === k ? 'on' : ''}" data-way="${k}">
              <span class="row" style="gap:8px"><span class="rd"></span><b style="font-size:15px">${t}</b></span><span class="muted" style="font-size:13px;line-height:1.55">${d}</span></button>`).join('')}
        </div>
        ${wayHtml()}
      </section>
      ${rows ? previewHtml() : ''}`}
      ${footer()}`;
    bind();
  }

  function wayHtml() {
    if (way === 'paste') return `<div class="col" style="gap:8px">
      <textarea class="textarea" id="im-text" rows="9" style="font-size:14px;line-height:1.7;font-family:Pretendard,monospace" placeholder="예)\n2019.03.10\t주일\t롬 8:28-30\t하나님의 영원한 작정\t본 교회\n2019.03.13 수요 시편 23편 여호와는 나의 목자시니\n\n엑셀이나 한글 표에서 칸째로 복사해 붙여 넣어도 되고, 한 줄에 그냥 적어도 돼요.">${esc(text)}</textarea>
      <div class="faint" style="font-size:12.5px;line-height:1.7">· 첫 줄에 <b>날짜 · 종류 · 본문 · 제목 · 장소 · 시리즈 · 요약</b> 같은 머리글이 있으면 그대로 따라요.<br>· 머리글이 없어도 날짜 · 종류 · 본문은 모양을 보고 알아봐요. 남은 칸은 차례대로 제목 → 장소 → 요약이에요.</div>
      <div class="row wrap" style="gap:8px"><button type="button" class="btn brand" data-act="read" ${busy ? 'disabled' : ''}>${busy ? '<span class="spinner"></span>' : ''}읽어 보기</button><button type="button" class="btn ghost" data-act="example">보기 넣어 보기</button></div></div>`;
    if (way === 'excel') return `<div class="col" style="gap:10px">
      <label class="drop" id="im-drop" style="padding:26px">엑셀 파일(.xlsx)을 여기에 끌어 놓거나 <b style="color:var(--brand)">눌러서 고르세요</b><input type="file" id="im-file" accept=".xlsx,.csv,.tsv,.txt" hidden></label>
      <div class="row wrap" style="gap:10px;font-size:13px"><button type="button" class="btn sm" data-act="template">${svgIcon(IC.file)}서식 파일 받기</button>
        <span class="muted">받아서 한 줄에 설교 하나씩 채운 뒤 넣으세요. 이미 정리해 둔 엑셀이 있으면 첫 줄 머리글만 맞춰 주세요.</span></div>
      ${busy ? '<div class="row muted" style="gap:8px"><span class="spinner"></span>읽는 중…</div>' : ''}</div>`;
    return `<div class="col" style="gap:10px">
      <div class="row wrap" style="gap:10px"><button type="button" class="btn brand" data-act="folder">${svgIcon(IC.folder)}${folder ? '다른 폴더 고르기' : '원고 폴더 고르기'}</button>
        ${folder ? `<span class="chip" style="max-width:100%"><span class="ellipsis">${esc(folder)}</span></span><button type="button" class="btn sm" data-act="read" ${busy ? 'disabled' : ''}>${busy ? '<span class="spinner"></span>' : ''}다시 읽기</button>` : ''}</div>
      <div class="row wrap" style="gap:18px"><label class="check"><input type="checkbox" id="im-rec" ${recursive ? 'checked' : ''}>안쪽 폴더까지 (예: 2019년 / 2020년 폴더)</label>
        <label class="check"><input type="checkbox" id="im-attach" ${attach ? 'checked' : ''}>원고 파일을 설교 기록에 첨부</label></div>
      <div class="tile" style="font-size:12.5px;line-height:1.8;color:var(--muted)">이런 파일 이름을 알아봐요<br>
        <b style="color:var(--text)">20190310 주일 롬8장28-30 하나님의 영원한 작정.hwp</b><br><b style="color:var(--text)">2019.03.13_수요_시편23편_여호와는 나의 목자.docx</b><br>
        <b style="color:var(--text)">2019년 폴더 / 0317 주일오후 엡1장3-14절 창세 전에 택하사.pdf</b> (날짜에 해가 없으면 폴더 이름의 해를 써요)<br>
        한글 · 워드 · PDF · Pages · 텍스트 · 녹음 · 영상 파일을 찾아요. 원래 파일은 그대로 두고, 앱 안에 복사해서 첨부해요.</div></div>`;
  }

  function previewHtml() {
    const on = rows.filter((r) => r.on);
    const issues = rows.filter((r) => issue(r));
    const dups = rows.filter((r) => r.dup);
    const shown = rows.map((r, i) => [r, i]).filter(([r]) => !onlyIssues || issue(r) || r.dup);
    const hasFile = rows.some((r) => r.fileName);
    return `<section class="card" aria-label="미리 보기">
      <div class="row wrap" style="gap:10px"><div class="step"><span class="k">2</span><h2>미리 보고 고치기</h2></div><span class="grow"></span>
        <span class="chip sm">모두 ${rows.length}줄</span><span class="chip sm" style="background:var(--brand-soft);color:var(--brand)">등록할 것 ${on.length}편</span>
        ${issues.length ? `<span class="chip sm" style="background:var(--warn-bg);color:var(--warn)">확인 필요 ${issues.length}</span>` : ''}
        ${dups.length ? `<span class="chip sm">이미 있음 ${dups.length} · 꺼 둠</span>` : ''}</div>
      <div class="row wrap tile" style="gap:10px;font-size:13px">
        <span class="muted">한꺼번에 바꾸기</span>
        <input class="input" id="im-place" placeholder="장소" style="width:160px;height:34px;font-size:13px"><button type="button" class="btn xs" data-act="setplace">장소 모두 바꾸기</button>
        <select class="select" id="im-status" style="width:120px;height:34px;font-size:13px">${STATUSES.map((s) => `<option ${s === '설교함' ? 'selected' : ''}>${s}</option>`).join('')}</select><button type="button" class="btn xs" data-act="setstatus">상태 모두 바꾸기</button>
        <span class="grow"></span>
        <label class="check" style="font-size:13px"><input type="checkbox" id="im-only" ${onlyIssues ? 'checked' : ''}>확인할 줄만 보기</label>
        <button type="button" class="btn xs" data-act="all">모두 고르기</button><button type="button" class="btn xs" data-act="none">모두 빼기</button>
      </div>
      <div style="overflow-x:auto;max-height:62vh;overflow-y:auto;border:1px solid var(--line);border-radius:12px">
        <table class="simple" style="min-width:${hasFile ? 1120 : 980}px">
          <thead style="position:sticky;top:0;background:var(--panel);z-index:1"><tr><th style="width:34px"></th><th style="width:150px">날짜</th><th style="width:118px">종류</th><th style="width:170px">본문</th><th>제목</th><th style="width:140px">장소</th>${hasFile ? '<th style="width:170px">원고 파일</th>' : '<th style="width:170px">읽은 글</th>'}</tr></thead>
          <tbody>${shown.map(([r, i]) => rowHtml(r, i, hasFile)).join('') || '<tr><td colspan="7" class="muted" style="padding:18px">확인할 줄이 없어요.</td></tr>'}</tbody>
        </table></div>
      <div class="row wrap" style="gap:10px;padding-top:4px">
        <button type="button" class="btn brand" data-act="commit" ${on.length && !busy ? '' : 'disabled'}>${busy ? '<span class="spinner"></span>' : ''}${on.length}편 등록하기</button>
        <span class="muted" style="font-size:13px">날짜가 없는 줄은 등록할 수 없어요. 본문을 알아보지 못한 줄은 적은 그대로 남겨요. 절기는 날짜를 보고 저절로 정해요.</span>
      </div></section>`;
  }

  function rowHtml(r, i, hasFile) {
    const iss = issue(r);
    const bg = !r.on ? 'opacity:.5' : '';
    return `<tr data-i="${i}" style="${bg}">
      <td><input type="checkbox" data-f="on" ${r.on ? 'checked' : ''} aria-label="${i + 1}번째 줄 등록" style="width:16px;height:16px;accent-color:var(--brand)"></td>
      <td><input class="input" type="date" data-f="date" value="${esc(r.date)}" style="height:34px;font-size:13px;padding:0 8px;${iss === '날짜' ? 'border-color:var(--warn)' : ''}"></td>
      <td><select class="select" data-f="kind" style="height:34px;font-size:13px;padding:0 26px 0 8px;background-position:calc(100% - 14px) 15px,calc(100% - 9px) 15px">${KINDS.map((k) => `<option ${k === r.kind ? 'selected' : ''}>${k}</option>`).join('')}</select>
        ${KIND_DETAIL[r.kind] ? `<input class="input" data-f="kindOther" value="${esc(r.kindOther || '')}" placeholder="이름" style="height:30px;font-size:12px;margin-top:4px;padding:0 8px">` : ''}</td>
      <td><input class="input" data-f="passageText" value="${esc(r.passageText)}" style="height:34px;font-size:13px;padding:0 8px;${iss === '본문' ? 'border-color:var(--warn)' : ''}">
        <div style="font-size:11px;margin-top:3px" class="${r.passageText ? (r.refOK ? 'ok-text' : 'warn-text') : 'faint'}">${r.passageText ? (r.refOK ? '✓ ' + esc(r.ref) : '본문을 알아보지 못했어요') : '본문 없음'}</div></td>
      <td><input class="input" data-f="title" value="${esc(r.title)}" placeholder="제목 없음" style="height:34px;font-size:13px;padding:0 8px">
        ${r.dup ? `<div class="warn-text" style="font-size:11px;margin-top:3px">${esc(r.dup)}</div>` : r.warn ? `<div class="warn-text" style="font-size:11px;margin-top:3px">${esc(r.warn)}</div>` : ''}</td>
      <td><input class="input" data-f="place" value="${esc(r.place)}" style="height:34px;font-size:13px;padding:0 8px"></td>
      <td class="faint" style="font-size:12px;line-height:1.45;word-break:break-all">${esc(hasFile ? (r.src || r.fileName) : r.src)}</td></tr>`;
  }

  function doneHtml() {
    return `<section class="card" style="align-items:flex-start">
      <h2 style="font-size:20px">${done.count}편을 등록했어요</h2>
      <p class="muted" style="margin:0;line-height:1.7">${done.files ? '원고 파일 ' + done.files + '개를 함께 첨부했어요. ' : ''}${done.failed && done.failed.length ? '첨부하지 못한 파일: ' + esc(done.failed.slice(0, 5).join(', ')) + (done.failed.length > 5 ? ' 외' : '') + '. ' : ''}설교 지도 · 달력 · 기록에 바로 보여요. 잘못 넣었다면 지금 되돌릴 수 있어요.</p>
      <div class="row wrap" style="gap:8px"><a class="btn brand" href="#records">설교 기록에서 보기</a><a class="btn" href="#map">설교 지도</a>
        <button type="button" class="btn" data-act="more">더 등록하기</button><button type="button" class="btn ghost danger" data-act="undo">방금 등록한 ${done.count}편 되돌리기</button></div>
    </section>`;
  }

  function bind() {
    el.onclick = async (e) => {
      const t = e.target.closest('button');
      if (!t) return;
      if (t.dataset.way) { way = t.dataset.way; rows = null; return draw(); }
      switch (t.dataset.act) {
        case 'example': text = EXAMPLE; draw(); break;
        case 'read': readNow(); break;
        case 'template': try { await api('/api/import/template', {}); toast('서식 파일을 저장했어요 · Finder 에서 보여 드릴게요'); } catch (err) { fail(err); } break;
        case 'folder': {
          const p = await chooseFolder('설교 원고 파일이 있는 폴더를 골라 주세요', folder);
          if (p) { folder = p; readNow(); }
          break;
        }
        case 'setplace': { const v = $('#im-place', el).value.trim(); rows.forEach((r) => { r.place = v; }); draw(); toast('장소를 모두 바꿨어요'); break; }
        case 'setstatus': { const v = $('#im-status', el).value; rows.forEach((r) => { r.status = v; }); toast('상태를 모두 ‘' + v + '’로 했어요'); break; }
        case 'all': rows.forEach((r) => { r.on = !!r.date; }); draw(); break;
        case 'none': rows.forEach((r) => { r.on = false; }); draw(); break;
        case 'commit': commit(); break;
        case 'more': done = null; rows = null; draw(); break;
        case 'undo': undo(); break;
      }
    };
    const ta = $('#im-text', el); if (ta) ta.oninput = (e) => { text = e.target.value; };
    const rec = $('#im-rec', el); if (rec) rec.onchange = (e) => { recursive = e.target.checked; if (folder) readNow(); };
    const at = $('#im-attach', el); if (at) at.onchange = (e) => { attach = e.target.checked; };
    const only = $('#im-only', el); if (only) only.onchange = (e) => { onlyIssues = e.target.checked; draw(); };
    const drop = $('#im-drop', el);
    if (drop) {
      drop.addEventListener('dragover', (e) => { e.preventDefault(); drop.classList.add('over'); });
      drop.addEventListener('dragleave', () => drop.classList.remove('over'));
      drop.addEventListener('drop', (e) => { e.preventDefault(); drop.classList.remove('over'); if (e.dataTransfer.files[0]) readFile(e.dataTransfer.files[0]); });
      $('#im-file', el).onchange = (e) => { if (e.target.files[0]) readFile(e.target.files[0]); };
    }
    // 표 칸 고치기
    const tb = el.querySelector('tbody');
    if (tb) {
      tb.addEventListener('change', async (e) => {
        const tr = e.target.closest('tr[data-i]'); if (!tr) return;
        const r = rows[+tr.dataset.i], f = e.target.dataset.f;
        if (f === 'on') { r.on = e.target.checked; if (r.on && !r.date) { r.on = false; e.target.checked = false; toast('날짜를 먼저 정해 주세요', { err: true }); } return draw(); }
        r[f] = e.target.value;
        if (f === 'date' && r.date && r.warn && r.warn.startsWith('날짜')) r.warn = '';
        if (f === 'date' && r.date && !r.dup) r.on = true;
        if (f === 'passageText') {
          try { const ps = (await api('/api/parse', { text: r.passageText })).passages; r.refOK = ps.length > 0 && ps.every((p) => p.book > 0); r.ref = ps.filter((p) => p.book > 0).map((p) => p.ref).join('; '); } catch (err) { /* 그대로 */ }
        }
        if (f === 'kind' || f === 'passageText' || f === 'date') { const m = $('#main'); const top = m.scrollTop; const box = el.querySelector('tbody').closest('div'); const st = box.scrollTop; draw(); m.scrollTop = top; el.querySelector('tbody').closest('div').scrollTop = st; }
      });
    }
  }

  function accept(list) {
    rows = list.map((r) => ({ ...r, on: !!r.date && !r.dup }));
    onlyIssues = false;
  }

  async function readNow() {
    if (way === 'paste' && !text.trim()) { toast('먼저 표를 붙여 넣어 주세요', { err: true }); return; }
    busy = true; draw();
    try {
      const r = way === 'folder' ? await api('/api/import/parse', { mode: 'folder', path: folder, recursive }) : await api('/api/import/parse', { mode: 'text', text });
      accept(r.rows);
    } catch (e) { fail(e); }
    busy = false; draw();
  }

  async function readFile(f) {
    const fd = new FormData();
    fd.append('file', f, f.name);
    busy = true; draw();
    try { accept((await api('/api/import/parse', fd)).rows); } catch (e) { fail(e); }
    busy = false; draw();
  }

  async function commit() {
    const list = rows.filter((r) => r.on);
    if (!list.length) return;
    const noDate = list.filter((r) => !r.date);
    if (noDate.length) { toast('날짜가 없는 줄이 있어요', { err: true }); return; }
    if (!(await confirmBox(list.length + '편을 등록할까요?', '등록하기 전에 지금 기록을 백업해 두고, 등록한 뒤에도 바로 되돌릴 수 있어요.', '등록하기'))) return;
    busy = true; draw();
    try {
      const body = list.map((r) => ({ ...r, season: r.season || seasonOption(seasonOf(parseISO(r.date))) }));
      done = await api('/api/import/commit', { rows: body, attach: way === 'folder' && attach });
      rows = null;
      await loadData();
    } catch (e) { fail(e); }
    busy = false; draw();
  }

  async function undo() {
    if (!(await confirmBox('방금 등록한 설교를 되돌릴까요?', `방금 등록한 ${done.count}편${done.files ? '과 첨부한 원고' : ''}를 지워요. 원래 파일은 그대로예요.`, '되돌리기', true))) return;
    try { const r = await api('/api/import/undo', { batch: done.batch }); done = null; await loadData(); draw(); toast(r.count + '편을 되돌렸어요'); } catch (e) { fail(e); }
  }

  return { render: (page) => render(page) };
})();
