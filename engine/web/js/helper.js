/* 설교 도우미 — 본문을 정하면 서재 PDF 에서 그 본문을 다룬 부분을 찾아, 그 자료를 근거로 정리해요 */
'use strict';

const HELPER_TABS = [['sum', '정리 · 요약'], ['str', '구조'], ['med', '묵상'], ['outline', '설교 개요'], ['ref', '개혁주의 관점']];

Screens.helper = (() => {
  let el = null;
  let ref = '', bibleText = '', sid = '', tab = 'outline', note = '';
  let hits = null;          // 서재에서 찾은 부분
  let off = {};             // 체크를 끈 자료 (doc:from)
  let extra = [];           // 직접 고른 자료
  let R = {};               // 탭 → 결과
  let job = null;           // 지금 만드는 중인 작업
  let pasteOpen = false;
  let copyAI = '';         // 구독 방식으로 요청문을 복사해 간 AI
  let sermon = null;
  let finding = false;

  async function render(page, q, changed) {
    el = page;
    if (changed) {
      if (q.sid && q.sid !== sid) {
        sid = q.sid; R = {}; hits = null; off = {}; extra = [];
        try {
          sermon = await api('/api/sermon?id=' + encodeURIComponent(sid));
          ref = refText(sermon) || sermon.passageText || '';
          bibleText = sermon.bibleText || '';
          for (const [k, v] of Object.entries(sermon.helper || {})) R[k] = v;
        } catch (e) { sid = ''; sermon = null; fail(e); }
      } else if (!q.sid && sid) { sid = ''; sermon = null; }
      if (q.ref && !q.sid) { ref = q.ref; hits = null; R = {}; }
      if (q.tab) tab = q.tab;
    }
    if (!S.lib) await loadLib();
    draw();
    if (ref && hits === null && S.lib && S.lib.folder && !finding) find();
  }

  async function loadLib() { try { S.lib = await api('/api/library'); } catch (e) { /* 다음에 */ } }
  const mode = () => S.settings.claudeMode === 'copy' ? 'copy' : 'api';

  function draw() {
    const L = S.lib || { docs: [] };
    const m = mode();
    el.innerHTML = `
      <div class="head" style="align-items:flex-end">
        <div class="t"><h1>설교 도우미</h1><p class="sub">본문을 정하면 내 서재의 PDF 에서 그 본문을 다룬 부분을 찾아, 그 자료를 근거로 정리해 드려요.</p></div>
        <label class="field" style="flex:0 1 280px">본문<input class="input" id="h-ref" value="${esc(ref)}" placeholder="예: 롬 8:28-30" style="font-size:16px;font-weight:700"></label>
        <button type="button" class="btn brand" data-act="find" style="height:42px">${finding ? '<span class="spinner"></span>' : ''}서재에서 찾기</button>
        <a class="btn" href="#settings" style="height:42px;font-size:13px;color:var(--muted)"><span class="dot" style="background:${m === 'api' ? (aiInfo().hasKey ? 'var(--ready)' : 'var(--prep)') : 'var(--sat)'}"></span>${esc(aiName())} · ${m === 'api' ? '앱에서 바로' : '구독(복사)'}</a>
      </div>
      ${sermon ? `<div class="row wrap tile" style="gap:8px;font-size:13px;background:var(--panel);border:1px solid var(--line)"><span class="muted">연결된 설교</span><b>${fmtDate(sermon.date, true)} ${esc(kindLabel(sermon))}</b><span>${esc(sermon.title || '제목 미정')}</span><span class="grow"></span><a class="link" href="#sermon?id=${sermon.id}">설교 기록 열기</a></div>` : ''}
      <div class="split">
        <aside class="aside" style="flex:1 1 340px" aria-label="서재에서 찾은 부분">${sourcesHtml(L)}</aside>
        <section class="main card" style="flex:999 1 560px;padding:0;gap:0" aria-label="도우미 결과">
          <div class="tabs" role="tablist">${HELPER_TABS.map(([k, l]) => `<button type="button" role="tab" data-tab="${k}" class="${k === tab ? 'on' : ''}" aria-selected="${k === tab}">${l}${R[k] ? '<span class="dot"></span>' : ''}</button>`).join('')}</div>
          <div id="h-result" style="padding:20px 22px;display:flex;flex-direction:column;gap:14px"></div>
          <div id="h-actions" class="row wrap" style="gap:8px;padding:14px 22px;border-top:1px solid var(--line);margin-top:auto"></div>
        </section>
      </div>${footer()}`;
    drawResult();
    $('#h-ref', el).addEventListener('keydown', (e) => { if (e.key === 'Enter') find(); });
    $('#h-ref', el).addEventListener('input', (e) => { ref = e.target.value; });
    el.onclick = onClick;
  }

  function sourcesHtml(L) {
    if (!L.folder) {
      return `<section class="card"><h2>서재</h2><p class="muted" style="margin:0;line-height:1.65;font-size:13px">주석 · 강해 · 신학서 PDF 를 모아 둔 폴더를 골라 주세요. 앱이 PDF 의 글자를 맥 안에서 읽어 두고(스캔한 PDF 도 글자 인식), 본문을 다룬 쪽을 찾아 드려요.</p>
        <button type="button" class="btn brand" data-act="folder">${svgIcon(IC.folder)}서재 폴더 고르기</button></section>`;
    }
    const busy = L.docs.filter((d) => d.status === 'reading' || d.status === 'waiting');
    const reading = L.docs.find((d) => d.status === 'reading');
    const srcs = allSources();
    const picked = srcs.filter((s) => !off[s.key]).length;
    return `<section class="card">
      <div class="ch"><h2>서재에서 찾은 부분</h2><span class="muted" style="font-size:12px">${srcs.length ? picked + ' / ' + srcs.length + '곳 사용' : ''}</span></div>
      ${busy.length ? `<div class="note" style="background:var(--panel2)"><span class="spinner" style="width:12px;height:12px;vertical-align:-2px;margin-right:6px"></span>PDF 를 읽는 중이에요 · 남은 자료 ${busy.length}개${reading ? `<br><span class="muted">${esc(reading.name)} ${reading.prog ? reading.prog + '쪽' : ''}</span>` : ''}</div>` : ''}
      ${hits === null ? `<div class="empty">${ref ? '‘서재에서 찾기’를 누르면 이 본문을 다룬 쪽을 찾아요.' : '본문을 적고 ‘서재에서 찾기’를 눌러 주세요.'}</div>`
        : srcs.length ? srcs.map((s) => `<label class="src ${off[s.key] ? 'off' : ''}">
            <input type="checkbox" data-src="${esc(s.key)}" ${off[s.key] ? '' : 'checked'}>
            <span class="col grow" style="gap:3px;min-width:0"><b style="font-size:14px;line-height:1.35">${esc(s.name)}</b>
              <span class="muted" style="font-size:12px">${s.from}–${s.to}쪽${s.hits ? ' · 본문 표기 ' + s.hits + '곳' : s.manual ? ' · 직접 고름' : ''}</span>
              ${s.snippet ? `<span class="faint" style="font-size:12px;line-height:1.5">${esc(s.snippet)}</span>` : ''}
              <span class="row" style="gap:10px"><button type="button" class="link" style="font-size:12px" data-peek="${esc(s.doc)}" data-p="${s.from}">쪽 보기</button>${s.manual ? `<button type="button" class="link" style="font-size:12px;color:var(--muted)" data-unpick="${esc(s.key)}">빼기</button>` : ''}</span></span>
          </label>`).join('') : '<div class="empty">서재에서 이 본문을 다룬 쪽을 찾지 못했어요. 아래 ‘자료 직접 고르기’로 쪽을 정해 넣을 수 있어요.</div>'}
      <div class="row wrap" style="gap:8px;padding-top:4px">
        <button type="button" class="btn sm grow" data-act="pick" style="border-style:dashed">+ 자료 직접 고르기</button>
        <button type="button" class="btn sm" data-act="library">서재 ${L.total || 0}권</button>
      </div>
      <p class="faint" style="margin:0;font-size:12px;line-height:1.6">PDF 글자는 맥 안에서 읽어 둬요. AI 에게는 체크한 부분만 보내요.</p>
    </section>
    <section class="card tight"><label class="field">이번에만 덧붙일 말 <span class="faint" style="font-weight:600">(선택)</span>
      <textarea class="textarea" id="h-note" rows="2" style="min-height:56px" placeholder="예: 청년부 대상이라 적용을 구체적으로">${esc(note)}</textarea></label></section>`;
  }

  function allSources() {
    const out = [];
    for (const h of hits || []) out.push({ key: h.doc + ':' + h.from, doc: h.doc, name: h.name, from: h.from, to: h.to, hits: h.hits, snippet: h.snippet });
    for (const x of extra) out.push({ ...x, key: x.doc + ':' + x.from + ':m', manual: true });
    return out;
  }
  const pickedSources = () => allSources().filter((s) => !off[s.key]).map((s) => ({ doc: s.doc, from: s.from, to: s.to }));

  async function find() {
    const r = ($('#h-ref', el) || {}).value;
    if (r !== undefined) ref = r.trim();
    if (!ref) { toast('본문을 먼저 적어 주세요', { err: true }); return; }
    if (!S.lib || !S.lib.folder) { toast('먼저 서재 폴더를 골라 주세요', { err: true }); return; }
    finding = true; draw();
    try {
      const res = await api('/api/library/find', { ref });
      hits = res.hits; off = {};
      hits.forEach((h, i) => { if (i >= 6) off[h.doc + ':' + h.from] = true; });
    } catch (e) { fail(e); hits = hits || []; }
    finding = false; draw();
  }

  function shortName(n) { const t = n.replace(/\.pdf$/i, '').trim(); return t.length > 10 ? t.slice(0, 9) + '…' : t; }
  function citeHtml(srcMap, c) {
    const s = srcMap[c.id];
    if (!s) return `<button type="button" class="cite" disabled>${esc(c.id)}${c.page ? ' ' + c.page : ''}</button>`;
    return `<button type="button" class="cite" data-peek="${esc(s.doc)}" data-p="${c.page || s.from}" title="${esc(s.name)} ${c.page || s.from}쪽 보기">${esc(shortName(s.name))} ${c.page || ''}</button>`;
  }

  function drawResult() {
    const box = $('#h-result', el), act = $('#h-actions', el);
    const N = aiName();
    if (!box) return;
    const res = R[tab];
    const label = (HELPER_TABS.find((x) => x[0] === tab) || [])[1];
    const refShown = (res && res.ref) || ref || '본문';
    let body = '';
    if (job && job.tab === tab && job.status === 'running') {
      body = `<div class="row" style="gap:12px;padding:30px 0;justify-content:center;color:var(--muted)"><span class="spinner"></span>${esc(aiName(job.ai))}${aiGap(aiName(job.ai))}가 자료를 읽고 정리하고 있어요… 보통 20초 ~ 1분 걸려요</div>`;
    } else if (res && res.mode === 'copy') {
      const srcMap = Object.fromEntries((res.sources || []).map((s) => [s.id, s]));
      body = `<div class="md">${mdToHtml(res.text || '', srcMap)}</div>`;
    } else if (res) {
      const srcMap = Object.fromEntries((res.sources || []).map((s) => [s.id, s]));
      const isOutline = tab === 'outline';
      body = `${res.thesis ? `<div class="thesis"><div class="l">중심 명제</div><div class="s">${esc(res.thesis)}</div></div>` : ''}
        <ol style="margin:0;padding:0;list-style:none;display:flex;flex-direction:column;gap:10px">${(res.items || []).map((it, i) => `<li class="result-item">
          ${isOutline ? '' : `<span class="n">${i + 1}</span>`}
          <div class="grow col" style="gap:7px;min-width:0"><div class="b">${it.head ? `<b>${esc(it.head)}</b>` : ''}${esc(it.body || '')}</div>
            <div class="row wrap" style="gap:5px">${(it.sources || []).map((c) => citeHtml(srcMap, c)).join('')}${it.outside ? '<button type="button" class="cite out" title="서재 자료에 없던 내용이에요. 꼭 확인해 주세요.">자료 밖</button>' : ''}</div></div></li>`).join('')}</ol>
        ${res.caution ? `<div class="note"><b>참고</b> · ${esc(res.caution)}</div>` : ''}`;
    } else {
      body = `<div class="empty" style="text-align:center;padding:30px">${mode() === 'api' ? `‘${esc(N)}${aiGap(N)}로 만들기’를 누르면 고른 자료를 근거로 ${label}을(를) 만들어요.` : `‘${esc(N)}${aiGap(N)}에 복사해서 열기’를 누르면 요청문과 고른 자료가 복사되고 ${esc(N)}${aiGap(N)}가 열려요. 받은 답을 아래에 붙여 넣으세요.`}</div>`;
    }
    const pasteHtml = (mode() === 'copy' || pasteOpen) ? `<label class="field" id="pastebox" style="${pasteOpen || mode() === 'copy' ? '' : 'display:none'}">${esc(aiName(copyAI || curAI()))} 에서 받은 답 붙여 넣기
        <textarea class="textarea" id="h-paste" rows="6" placeholder="${esc(aiName(copyAI || curAI()))} 에서 받은 답을 그대로 붙여 넣고 ‘붙여 넣은 답 저장’을 누르세요">${res && res.mode === 'copy' ? esc(res.text) : ''}</textarea>
        <span class="row" style="justify-content:flex-end"><button type="button" class="btn sm ink" data-act="paste">붙여 넣은 답 저장</button></span></label>` : '';
    const journal = tab === 'med' ? `<label class="field">나의 묵상 기록 ${sermon ? '' : '<span class="faint" style="font-weight:600">· 설교 기록에 저장할 때 함께 저장돼요</span>'}
        <textarea class="textarea serif" id="h-med" rows="4" placeholder="말씀 앞에서 받은 것을 적어 두세요">${esc(sermon ? sermon.meditation || '' : '')}</textarea></label>` : '';
    box.innerHTML = `<div class="row wrap" style="align-items:baseline;gap:10px"><h2 class="serif" style="font-size:22px;font-weight:700">${esc(refShown)} · ${esc(label)}</h2>
        <span class="faint" style="font-size:12px">${res && res.at ? aiTag(res.ai) + (res.mode === 'copy' ? '구독으로 받은 답' : '앱에서 바로') + ' · ' + esc(res.at.replace('T', ' ').slice(0, 16)) : ''}</span>
        ${res && res.prev ? `<button type="button" class="link" data-act="swap" style="font-size:12px" title="같은 본문을 다른 AI 로 받은 답과 번갈아 볼 수 있어요">${esc(aiName(res.prev.ai || 'claude'))}${aiGap(aiName(res.prev.ai || 'claude'))}의 답 보기</button>` : ''}</div>
      ${body}${journal}${pasteHtml}
      <div class="row wrap" style="gap:14px;font-size:12px;color:var(--muted)">
        <span class="row" style="gap:6px"><span class="cite" style="height:18px;display:inline-flex;align-items:center">칼빈 313</span>서재 자료 · 누르면 그 쪽이 열려요</span>
        <span class="row" style="gap:6px"><span class="cite out" style="height:18px;display:inline-flex;align-items:center">자료 밖</span>AI 가 덧붙인 내용 · 꼭 확인하세요</span></div>`;
    const busy = job && job.status === 'running';
    const both = S.settings.showBoth;
    act.innerHTML = `
      ${res ? `<button type="button" class="btn brand" data-act="saveto" ${busy ? 'disabled' : ''}>설교 기록에 저장</button>` : ''}
      ${res && tab === 'outline' && res.mode !== 'copy' ? `<button type="button" class="btn" data-act="tooutline">중심 명제 · 대지로 옮기기</button>` : ''}
      ${mode() === 'api' ? `<button type="button" class="btn ${res ? '' : 'brand'}" data-act="run" ${busy ? 'disabled' : ''}>${res ? '다시 만들기' : esc(N) + ' 로 만들기'}</button>` : `<button type="button" class="btn ${res ? '' : 'brand'}" data-act="copy">${esc(N)}${aiGap(N)}에 복사해서 열기</button>`}
      ${res ? `<button type="button" class="btn" data-act="copyres">복사</button>` : ''}
      <button type="button" class="btn" data-act="otherai" ${busy ? 'disabled' : ''} title="같은 본문 · 같은 자료로 다른 AI 의 답도 받아 비교해 보세요">다른 AI 로도 받아 보기</button>
      <span class="grow"></span>
      ${both ? (mode() === 'api' ? `<button type="button" class="btn ghost" data-act="copy" style="font-size:13px;color:var(--muted)" title="구독 방식: 요청문과 자료를 복사하고 ${esc(N)}${aiGap(N)}를 열어요">${esc(N)}${aiGap(N)}에 복사해서 열기</button>`
        : `<button type="button" class="btn ghost" data-act="run" style="font-size:13px;color:var(--muted)">앱에서 바로 만들기</button>`) : ''}`;
  }

  // 아주 간단한 마크다운 → 화면 (구독 방식으로 받은 답)
  function mdToHtml(md, srcMap) {
    const inline = (s) => {
      let h = esc(s);
      h = h.replace(/\*\*(.+?)\*\*/g, '<b>$1</b>').replace(/(^|[^*])\*([^*]+)\*/g, '$1<i>$2</i>');
      h = h.replace(/\[(S\d+)\s*(\d+)?\s*쪽?\]/g, (m0, id, p) => citeHtml(srcMap, { id, page: p ? +p : 0 }));
      h = h.replace(/\(자료\s?밖\)/g, '<button type="button" class="cite out">자료 밖</button>');
      return h;
    };
    const lines = md.replace(/\r/g, '').split('\n');
    let out = '', list = '';
    const close = () => { if (list) { out += `</${list}>`; list = ''; } };
    for (const ln of lines) {
      let m;
      if ((m = /^\s*#{1,6}\s+(.*)$/.exec(ln))) { close(); out += `<h3>${inline(m[1])}</h3>`; }
      else if ((m = /^\s*[-*•·]\s+(.*)$/.exec(ln))) { if (list !== 'ul') { close(); out += '<ul>'; list = 'ul'; } out += `<li>${inline(m[1])}</li>`; }
      else if ((m = /^\s*\d+[.)]\s+(.*)$/.exec(ln))) { if (list !== 'ol') { close(); out += '<ol>'; list = 'ol'; } out += `<li>${inline(m[1])}</li>`; }
      else if (!ln.trim()) { close(); }
      else { close(); out += `<p>${inline(ln)}</p>`; }
    }
    close();
    return out;
  }

  async function onClick(e) {
    const t = e.target.closest('button, a');
    if (!t) return;
    const d = t.dataset;
    if (d.tab) { tab = d.tab; $$('.tabs button', el).forEach((b) => { b.classList.toggle('on', b === t); b.setAttribute('aria-selected', b === t); }); drawResult(); return; }
    if (d.peek) { peek(d.peek, +d.p); return; }
    if (d.unpick) { extra = extra.filter((x) => x.doc + ':' + x.from + ':m' !== d.unpick); draw(); return; }
    switch (d.act) {
      case 'find': find(); break;
      case 'folder': setFolder(); break;
      case 'library': libraryModal(); break;
      case 'pick': pickManual(); break;
      case 'run': run(); break;
      case 'copy': copyMode(); break;
      case 'paste': savePaste(); break;
      case 'saveto': saveTo(); break;
      case 'tooutline': toOutline(); break;
      case 'copyres': copyResult(); break;
      case 'otherai': otherAI(); break;
      case 'swap': swapPrev(); break;
    }
  }
  // 체크 상자
  document.addEventListener('change', (e) => {
    if (S.route.name !== 'helper') return;
    const c = e.target.closest('[data-src]');
    if (c) { off[c.dataset.src] = !c.checked; c.closest('.src').classList.toggle('off', !c.checked); const n = allSources().filter((s) => !off[s.key]).length; const h = c.closest('.card').querySelector('.ch .muted'); if (h) h.textContent = n + ' / ' + allSources().length + '곳 사용'; }
    if (e.target.id === 'h-note') note = e.target.value;
  });
  document.addEventListener('input', (e) => { if (S.route.name === 'helper' && e.target.id === 'h-note') note = e.target.value; });

  function request(ai) {
    return { ref: ref || ($('#h-ref', el) || {}).value || '', tab, bibleText, sources: pickedSources(), note, ai: ai || curAI() };
  }

  const aiTag = (id) => esc(aiName(id || 'claude')) + ' · ';

  // 새 답을 넣을 때, 다른 AI 의 답이 있던 자리면 그 답을 ‘이전 답’으로 남겨 번갈아 볼 수 있게 해요
  function setResult(t, res) {
    const old = R[t];
    if (old) {
      const o = { ...old }; delete o.prev;
      res.prev = (old.ai || 'claude') !== (res.ai || 'claude') ? o : old.prev;
    }
    R[t] = res;
  }
  function swapPrev() {
    const cur = R[tab];
    if (!cur || !cur.prev) return;
    const now = { ...cur }; delete now.prev;
    R[tab] = { ...cur.prev, prev: now };
    drawResult();
    toast(`${aiName(R[tab].ai || 'claude')}${aiGap(aiName(R[tab].ai || 'claude'))}의 답을 보여 드려요 · 저장하면 지금 보이는 답이 앞에 남아요`);
  }

  async function otherAI() {
    const others = AI_LIST.filter(([k]) => k !== curAI());
    const v = await modal({
      title: '다른 AI 로도 받아 보기',
      body: `<p style="margin:0 0 12px;color:var(--muted);line-height:1.65">같은 본문 · 같은 자료 · 같은 기준으로 다른 AI 의 답을 받아요. 받은 뒤에는 결과 위의 ‘○○ 의 답 보기’로 번갈아 볼 수 있어요.</p>
        <div class="col" style="gap:8px">${others.map(([k, n, co]) => { const P = aiInfo(k); return `<div class="tile row wrap" style="gap:10px;padding:12px">
          <span class="dot" style="background:${AI_COLOR[k]}"></span><b>${n}</b><span class="faint" style="font-size:12px">${co}${P.hasKey ? ' · API 키 있음' : ''}</span><span class="grow"></span>
          ${P.hasKey ? `<button type="button" class="btn sm brand" data-v="api:${k}">앱에서 바로</button>` : ''}
          <button type="button" class="btn sm" data-v="copy:${k}">복사해서 ${n} 열기</button></div>`; }).join('')}</div>
        <p class="faint" style="margin:12px 0 0;font-size:12px">앱에서 바로 받으려면 설정 › 설교 도우미 · AI 연결에서 그 AI 의 API 키를 넣어 주세요.</p>`,
      buttons: [{ label: '닫기', value: null }],
      onOpen: (m, close) => m.addEventListener('click', (e) => { const b = e.target.closest('[data-v]'); if (b) close(b.dataset.v); }),
    });
    if (!v) return;
    const [how, k] = v.split(':');
    if (how === 'api') run(k); else copyMode(k);
  }

  async function run(aiOverride) {
    const ai = aiOverride || curAI(), N = aiName(ai);
    if (!aiInfo(ai).hasKey) {
      const r = await modal({ title: `${N} API 키가 아직 없어요`, body: `<p style="margin:0;color:var(--muted);line-height:1.65">앱에서 바로 만들려면 설정 › 설교 도우미 · AI 연결에서 ${N} API 키를 넣어 주세요. 키 없이 지금 쓰시는 ${N} 구독으로 하려면 ‘복사해서 열기’를 쓰세요.</p>`,
        buttons: [{ label: '닫기', value: '' }, { label: '복사해서 열기', value: 'copy' }, { label: '설정 열기', kind: 'brand', value: 'settings' }] });
      if (r === 'settings') go('settings'); if (r === 'copy') copyMode(ai);
      return;
    }
    if (!request().ref.trim()) { toast('본문을 먼저 적어 주세요', { err: true }); return; }
    if (!pickedSources().length && !(await confirmBox('고른 자료가 없어요', '서재 자료 없이 만들면 모든 내용이 ‘자료 밖’ 내용이 돼요. 그래도 만들까요?', '그래도 만들기'))) return;
    try {
      job = await api('/api/helper/run', request(ai));
      drawResult();
      const myTab = job.tab;
      while (job && job.status === 'running') {
        await new Promise((r) => setTimeout(r, 1500));
        job = await api('/api/helper/job?id=' + job.id);
      }
      if (job.status === 'done') { setResult(myTab, job.result); toast(`${N}${aiGap(N)}가 정리했어요 · ‘설교 기록에 저장’을 누르면 남겨져요`); }
      else fail(new Error(job.error || '만들지 못했어요'));
    } catch (e) { fail(e); }
    job = null;
    if (S.route.name === 'helper') { const cur = tab; draw(); tab = cur; }
  }

  async function copyMode(aiOverride) {
    const ai = aiOverride || curAI(), N = aiName(ai);
    try {
      const r = await api('/api/helper/prompt', request(ai));
      copyAI = ai;
      await copyText(r.text);
      pasteOpen = true;
      R[tab] = R[tab] && R[tab].mode === 'copy' ? R[tab] : R[tab];
      S._copySources = r.sources || [];
      drawResult();
      const ok = await modal({ title: '요청문을 복사했어요', body: `<p style="margin:0;line-height:1.7;color:var(--muted)">요청문과 고른 자료(${r.chars.toLocaleString()}자)가 복사됐어요.<br>① ${N}${aiGap(N)}가 열리면 새 대화에 <b>붙여 넣기(⌘V)</b> 하고 보내세요.<br>② 받은 답을 복사해서 도우미의 <b>‘${N}${aiGap(N)}에서 받은 답 붙여 넣기’</b> 칸에 붙여 넣으세요.</p>
        ${r.chars > 150000 ? '<div class="note"><b>자료가 많아요</b> · 한 번에 붙여 넣기 어려우면 자료 몇 개의 체크를 끄고 다시 복사해 주세요.</div>' : ''}`,
        buttons: [{ label: '닫기', value: false }, { label: `${N} 열기`, kind: 'brand', value: true }] });
      if (ok) openExternal(r.chatURL || AI_CHAT[ai]);
      setTimeout(() => { const p = $('#h-paste'); if (p) p.focus(); }, 100);
    } catch (e) { fail(e); }
  }

  function savePaste() {
    const t = ($('#h-paste', el) || {}).value || '';
    if (!t.trim()) { toast('받은 답을 먼저 붙여 넣어 주세요', { err: true }); return; }
    const srcs = (S._copySources && S._copySources.length) ? S._copySources : allSources().filter((s) => !off[s.key]).map((s, i) => ({ id: 'S' + (i + 1), doc: s.doc, name: s.name, from: s.from, to: s.to }));
    setResult(tab, { mode: 'copy', ai: copyAI || curAI(), text: t.trim(), sources: srcs, ref: ref, tab, at: new Date(Date.now() - new Date().getTimezoneOffset() * 60000).toISOString().slice(0, 16) });
    pasteOpen = false; copyAI = '';
    draw();
    toast('붙여 넣은 답을 정리했어요 · ‘설교 기록에 저장’을 누르면 남겨져요');
  }

  async function ensureSermon() {
    if (sid) return sid;
    const ps = (await api('/api/parse', { text: ref })).passages.filter((p) => p.book > 0);
    const near = sermonsOverlapping(ps, '').filter((s) => s.status !== '설교함').concat(sermonsOverlapping(ps, '').filter((s) => s.status === '설교함')).slice(0, 8);
    const r = await modal({
      title: '어느 설교 기록에 저장할까요?',
      body: `${near.length ? `<div class="col" style="gap:8px">${near.map((s) => `<label class="src"><input type="radio" name="tgt" value="${s.id}"><span class="col grow" style="gap:2px"><b>${fmtDate(s.date, true)} ${esc(kindLabel(s))} · ${esc(s.title || '제목 미정')}</b><span class="muted" style="font-size:12px">${esc(s._ref)} · ${esc(s.place || '')} · ${esc(s.status)}</span></span></label>`).join('')}</div>` : '<p class="muted" style="margin:0">이 본문으로 된 설교 기록이 아직 없어요.</p>'}
        <label class="src"><input type="radio" name="tgt" value="new" ${near.length ? '' : 'checked'}><span class="col grow" style="gap:2px"><b>새 설교 기록 만들기</b><span class="muted" style="font-size:12px">본문 ${esc(ref)} · 준비 중</span></span></label>`,
      buttons: [{ label: '취소', value: null }, { label: '저장', kind: 'brand', value: (m) => { const c = m.querySelector('input[name=tgt]:checked'); return c ? c.value : null; } }],
    });
    if (!r) return '';
    if (r === 'new') {
      const res = await api('/api/sermon/save', { sermon: { date: iso(today()), kind: '주일', passageText: ref, status: '준비 중', place: (S.settings.ui && S.settings.ui.homePlace) || '', bibleText, points: [], confessions: [], illus: [] } });
      sid = res.sermon.id;
    } else sid = r;
    sermon = await api('/api/sermon?id=' + sid);
    history.replaceState(null, '', '#helper?sid=' + sid + '&tab=' + tab);
    lastRouteKey = location.hash;
    return sid;
  }

  async function saveTo() {
    try {
      const id = await ensureSermon();
      if (!id) return;
      await api('/api/sermon/helper', { id, tab, result: R[tab] });
      const med = $('#h-med', el);
      if (med && sermon) {
        const s = await api('/api/sermon?id=' + id);
        s.meditation = med.value;
        await api('/api/sermon/save', { sermon: s });
      }
      await loadData();
      sermon = await api('/api/sermon?id=' + id);
      draw();
      toast('설교 기록에 저장했어요', { action: '설교 기록 열기', onAction: () => go('sermon', { id }) });
    } catch (e) { fail(e); }
  }

  async function toOutline() {
    const res = R.outline;
    if (!res) return;
    try {
      const id = await ensureSermon();
      if (!id) return;
      const s = await api('/api/sermon?id=' + id);
      const pts = (res.items || []).filter((it) => /^\s*[ⅠⅡⅢⅣⅤⅥⅦⅧⅨⅩ]|^\s*[IVX]+\.|^\s*\d+\./.test(it.head || '')).map((it) => (it.head || '').replace(/^\s*([ⅠⅡⅢⅣⅤⅥⅦⅧⅨⅩ]+|[IVX]+|\d+)\.?\s*/, '').replace(/\s*—\s*$/, '').trim()).filter(Boolean);
      if ((s.thesis || (s.points || []).length) && !(await confirmBox('중심 명제 · 대지를 바꿀까요?', '설교 기록에 이미 적어 둔 중심 명제 · 대지를 도우미의 개요로 바꿔요.', '바꾸기'))) return;
      s.thesis = res.thesis || s.thesis;
      if (pts.length) s.points = pts;
      await api('/api/sermon/save', { sermon: s });
      await api('/api/sermon/helper', { id, tab: 'outline', result: res });
      await loadData();
      toast('중심 명제와 대지를 설교 기록에 넣었어요', { action: '설교 기록 열기', onAction: () => go('sermon', { id }) });
    } catch (e) { fail(e); }
  }

  async function copyResult() {
    const res = R[tab];
    if (!res) return;
    let text;
    if (res.mode === 'copy') text = res.text;
    else {
      const srcMap = Object.fromEntries((res.sources || []).map((s) => [s.id, s]));
      text = (res.thesis ? '중심 명제: ' + res.thesis + '\n\n' : '') + (res.items || []).map((it, i) => `${tab === 'outline' ? '' : (i + 1) + '. '}${it.head || ''}${it.body || ''}${(it.sources || []).length ? ' [' + it.sources.map((c) => (srcMap[c.id] ? srcMap[c.id].name : c.id) + (c.page ? ' ' + c.page + '쪽' : '')).join(', ') + ']' : ''}${it.outside ? ' (자료 밖)' : ''}`).join('\n');
    }
    if (await copyText(text)) toast('복사했어요');
  }

  async function peek(doc, p) {
    let cur = p;
    const load = async (m) => {
      try {
        const r = await api(`/api/library/page?doc=${encodeURIComponent(doc)}&p=${cur}`);
        $('.mbody', m).innerHTML = `<div class="row" style="gap:8px"><b class="grow">${esc(r.name)}</b><button type="button" class="icon-btn" data-pp="-1" ${cur <= 1 ? 'disabled' : ''} aria-label="앞 쪽">${svgIcon(IC.left)}</button>
          <span class="num muted" style="font-size:13px">${cur} / ${r.pages}쪽</span><button type="button" class="icon-btn" data-pp="1" ${cur >= r.pages ? 'disabled' : ''} aria-label="다음 쪽">${svgIcon(IC.right)}</button></div>
          <div class="pagetext">${esc(r.text) || '<span class="faint">이 쪽에는 글자가 없어요</span>'}</div>
          <p class="faint" style="margin:0;font-size:12px">쪽 번호는 PDF 파일의 쪽 순서예요 (책에 인쇄된 쪽 번호와 다를 수 있어요).</p>`;
        $$('[data-pp]', m).forEach((b) => b.onclick = () => { cur += +b.dataset.pp; load(m); });
      } catch (e) { $('.mbody', m).innerHTML = `<div class="empty">${esc(e.message)}</div>`; }
    };
    const r = await modal({ title: '서재 자료', wide: true, body: '<div class="row"><span class="spinner"></span></div>', focus: false, onOpen: (m) => load(m),
      buttons: [{ label: 'PDF 열기', value: 'open' }, { label: '닫기', kind: 'brand', value: null }] });
    if (r === 'open') api('/api/library/open', { doc }).catch(fail);
  }

  async function setFolder() {
    const p = await chooseFolder('설교 서재(주석 · 강해 PDF)가 있는 폴더를 골라 주세요', S.lib && S.lib.folder);
    if (!p) return;
    try { S.lib = await api('/api/library/folder', { path: p }); toast('서재 폴더를 정했어요. PDF 를 읽기 시작해요'); draw(); } catch (e) { fail(e); }
  }

  async function pickManual() {
    const docs = (S.lib && S.lib.docs || []).filter((d) => d.status === 'ready');
    if (!docs.length) { toast('먼저 서재 폴더를 고르고 PDF 를 읽을 때까지 기다려 주세요', { err: true }); return; }
    const r = await modal({
      title: '자료 직접 고르기',
      body: `<label class="field">자료<select class="select" id="mp-doc">${docs.map((d) => `<option value="${d.id}">${esc(d.rel || d.name)} (${d.pages}쪽)</option>`).join('')}</select></label>
        <div class="row" style="gap:10px"><label class="field grow">처음 쪽<input class="input" id="mp-from" type="number" min="1" value="1"></label><label class="field grow">마지막 쪽<input class="input" id="mp-to" type="number" min="1" value="5"></label></div>
        <p class="faint" style="margin:0;font-size:12px">PDF 파일의 쪽 순서로 적어 주세요. 한 번에 30쪽까지 넣을 수 있어요.</p>`,
      buttons: [{ label: '취소', value: null }, { label: '넣기', kind: 'brand', value: (m) => ({ doc: $('#mp-doc', m).value, from: +$('#mp-from', m).value, to: +$('#mp-to', m).value }) }],
    });
    if (!r || !r.from) return;
    const d = docs.find((x) => x.id === r.doc);
    const from = Math.max(1, Math.min(r.from, d.pages)), to = Math.max(from, Math.min(r.to || from, d.pages, from + 29));
    extra.push({ doc: d.id, name: d.name, from, to });
    if (hits === null) hits = [];
    draw();
  }

  async function libraryModal() {
    const body = () => {
      const L = S.lib || { docs: [] };
      const st = (d) => d.status === 'ready' ? `<span class="ok-text">읽음 · ${d.pages}쪽${d.ocr ? ' · 글자 인식 ' + d.ocr + '쪽' : ''} · 본문 표기 ${d.refs}곳</span>`
        : d.status === 'reading' ? `<span class="muted" style="font-size:12px"><span class="spinner" style="width:10px;height:10px;vertical-align:-1px"></span> 읽는 중 ${d.prog || ''}</span>`
        : d.status === 'waiting' ? '<span class="faint" style="font-size:12px">차례를 기다리는 중</span>' : `<span class="warn-text">${esc(d.err || '읽지 못했어요')}</span>`;
      return `<div class="row wrap" style="gap:8px"><span class="chip" style="max-width:100%"><span class="ellipsis">${esc(L.folder || '폴더 없음')}</span></span><span class="grow"></span>
          <button type="button" class="btn sm" data-lm="folder">폴더 바꾸기</button><button type="button" class="btn sm" data-lm="rescan">다시 훑기</button></div>
        ${L.error ? `<div class="note"><b>문제</b> · ${esc(L.error)}</div>` : ''}
        <div class="muted" style="font-size:13px">PDF ${L.total || 0}개 · 읽음 ${L.ready || 0}개 · ${(L.pages || 0).toLocaleString()}쪽</div>
        <div class="col" style="gap:6px;max-height:52vh;overflow-y:auto">${(L.docs || []).map((d) => `<div class="tile row wrap" style="gap:8px">
          <span class="col grow" style="gap:2px;min-width:200px"><b style="font-size:13px">${esc(d.rel || d.name)}</b>${st(d)}</span>
          <select class="select" data-book="${d.id}" style="width:170px;height:32px;font-size:12px" title="이 자료가 주로 다루는 책 — 장 · 절만 적힌 쪽을 찾을 때 써요"><option value="0">${d.book && !d.bookOverride ? '자동: ' + esc(bookOf(d.book).name) : '자동 (여러 책)'}</option>${S.books.map((b) => `<option value="${b.n}" ${d.bookOverride === b.n ? 'selected' : ''}>${esc(b.name)}</option>`).join('')}</select>
          <button type="button" class="btn xs" data-redo="${d.id}">다시 읽기</button></div>`).join('') || '<div class="empty">폴더 안에 PDF 가 없어요.</div>'}</div>
        <p class="faint" style="margin:0;font-size:12px;line-height:1.6">‘주로 다루는 책’을 정해 두면(예: 칼빈 로마서 주석 → 로마서) ‘28절’처럼 책 이름 없이 적힌 쪽도 찾아요. 파일 이름에 책 이름이 있으면 저절로 정해져요.</p>`;
    };
    let timer;
    await modal({ title: '서재', wide: true, body: body(), focus: false, buttons: [{ label: '닫기', kind: 'brand', value: null }],
      onOpen: (m) => {
        const re = async () => { try { S.lib = await api('/api/library'); $('.mbody', m).innerHTML = body(); } catch (e) { /* 다음에 */ } };
        timer = setInterval(() => { if (!document.body.contains(m)) { clearInterval(timer); return; } if (S.lib && S.lib.busy) re(); }, 2000);
        m.addEventListener('click', async (e) => {
          const b = e.target.closest('[data-lm],[data-redo]'); if (!b) return;
          try {
            if (b.dataset.lm === 'folder') { const p = await chooseFolder('설교 서재 폴더를 골라 주세요', S.lib.folder); if (p) S.lib = await api('/api/library/folder', { path: p }); }
            if (b.dataset.lm === 'rescan') S.lib = await api('/api/library/rescan', {});
            if (b.dataset.redo) S.lib = await api('/api/library/rescan', { doc: b.dataset.redo });
            $('.mbody', m).innerHTML = body();
          } catch (err) { fail(err); }
        });
        m.addEventListener('change', async (e) => {
          const s = e.target.closest('[data-book]'); if (!s) return;
          try { S.lib = await api('/api/library/book', { doc: s.dataset.book, book: +s.value }); $('.mbody', m).innerHTML = body(); } catch (err) { fail(err); }
        });
      } });
    clearInterval(timer);
    hits = null;
    draw();
  }

  return {
    render,
    refreshLib: () => { if (el && S.route.name === 'helper' && !job) { const n = $('#h-note'); if (n) note = n.value; draw(); } },
    refresh: () => {},
  };
})();
