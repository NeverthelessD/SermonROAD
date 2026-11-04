/* 설정 — Claude 연결 · 개혁주의 기준 · 서재 · 화면 모드 · 업데이트 */
'use strict';

Screens.settings = (() => {
  async function render(el) {
    try { S.claude = await api('/api/claude'); } catch (e) { /* 그대로 */ }
    if (!S.lib) { try { S.lib = await api('/api/library'); } catch (e) { /* 그대로 */ } }
    const st = S.settings, c = S.claude || { hasKey: false, models: [], usage: {} };
    const m = st.claudeMode === 'copy' ? 'copy' : 'api';
    const L = S.lib || {};
    const u = S.update || {};
    el.innerHTML = `
      <h1>설정</h1>
      <div class="split">
        <div class="main">
          <section class="card" aria-label="Claude 연결">
            <div><h2 style="font-size:16px">설교 도우미 · Claude 연결</h2><p class="muted" style="margin:4px 0 0;font-size:13px">언제든 바꿀 수 있어요.</p></div>
            <div role="radiogroup" aria-label="연결 방법" style="display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:10px">
              ${[['api', '앱에서 바로 (API 키)', '버튼 하나로 도우미 결과가 앱 안에 바로 나와요. 여러 PDF 자료를 한꺼번에 보내기 좋아요.', '추천 · 쓴 만큼 요금'],
                 ['copy', '지금 쓰는 Claude 구독으로', '앱이 요청문과 자료를 준비해 복사해 주면, Claude 에 붙여 넣고 답을 다시 가져와요.', '추가 비용 없음 · 손이 조금 더 가요']].map(([k, t, d, tag]) =>
                `<button type="button" role="radio" aria-checked="${m === k}" class="radio-card ${m === k ? 'on' : ''}" data-mode="${k}">
                  <span class="row" style="gap:8px"><span class="rd"></span><b style="font-size:15px">${t}</b></span>
                  <span class="muted" style="font-size:13px;line-height:1.55">${d}</span><span style="font-size:12px;font-weight:700;color:${m === k ? 'var(--brand)' : 'var(--faint)'}">${tag}</span></button>`).join('')}
            </div>
            ${m === 'api' ? `<div class="tile col" style="gap:10px;padding:14px">
              <label class="field" for="apikey">API 키 ${c.hasKey ? `<span class="faint" style="font-weight:600">· 저장된 키 ${esc(c.masked)}</span>` : ''}</label>
              <div class="row wrap" style="gap:8px">
                <input class="input" id="apikey" type="password" autocomplete="off" placeholder="${c.hasKey ? '바꾸려면 새 키를 붙여 넣으세요' : 'sk-ant- 로 시작하는 키를 붙여 넣으세요'}" style="flex:1 1 260px">
                <button type="button" class="btn ink" data-act="savekey">저장</button>
                <button type="button" class="btn" data-act="testkey" ${c.hasKey ? '' : 'disabled'}>연결 시험</button>
                ${c.hasKey ? '<button type="button" class="btn ghost danger" data-act="delkey">키 지우기</button>' : ''}
              </div>
              <div class="row wrap" style="gap:14px;font-size:12px;color:var(--muted)">
                <span class="row" style="gap:6px"><span class="dot" style="background:${c.hasKey ? 'var(--ready)' : 'var(--line2)'}"></span>${c.hasKey ? '키가 있어요 · 키는 이 맥의 키체인에만 저장해요' : '아직 키가 없어요'}</span>
                <button type="button" class="link" data-act="howkey" style="font-size:12px">키 받는 방법</button></div>
              <label class="field">모델<select class="select" id="model">${(c.models || []).map((x) => `<option value="${esc(x.id)}" ${x.id === st.model ? 'selected' : ''}>${esc(x.label)}</option>`).join('')}</select></label>
              <div class="faint" style="font-size:12px;line-height:1.6">쓴 만큼 요금이 따로 나가요 (Anthropic 콘솔에서 확인). 이번 달 이 앱에서 ${(c.usage && c.usage.calls) || 0}번 불렀어요 · 보낸 글 ${((c.usage && c.usage.in) || 0).toLocaleString()} 토큰 · 받은 글 ${((c.usage && c.usage.out) || 0).toLocaleString()} 토큰.</div>
            </div>` : `<div class="tile" style="padding:14px;font-size:13px;line-height:1.7;color:var(--muted)"><b style="color:var(--text)">이렇게 써요</b> — 도우미에서 ‘Claude 에 복사해서 열기’를 누르면 요청문과 고른 자료가 복사되고 Claude 가 열려요. 붙여 넣고 받은 답을 도우미의 ‘Claude 의 답 붙여 넣기’ 칸에 넣으면 설교 기록에 저장할 수 있어요. 자료가 아주 많으면 체크를 몇 개 끄고 보내 주세요.</div>`}
            <label class="check"><input type="checkbox" id="showboth" ${st.showBoth ? 'checked' : ''}>도우미 화면에 다른 방법 버튼도 함께 보이기</label>
          </section>

          <section class="card" aria-label="개혁주의 기준">
            <div><h2 style="font-size:16px">개혁주의 관점 기준</h2><p class="muted" style="margin:4px 0 0;font-size:13px">설교 도우미가 늘 기준으로 삼을 문서예요.</p></div>
            <div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:8px">
              ${STANDARDS.map(([k, n]) => `<label class="tile check"><input type="checkbox" data-std="${k}" ${(st.standards || []).includes(k) ? 'checked' : ''}>${n}</label>`).join('')}</div>
            <label class="field">도우미에게 늘 덧붙일 말<textarea class="textarea" id="extra" rows="3">${esc(st.extraPrompt || '')}</textarea></label>
          </section>

          <section class="card" aria-label="서재" style="flex-direction:row;flex-wrap:wrap;align-items:center;gap:12px">
            <div style="flex:1 1 280px;min-width:0"><h2 style="font-size:16px">서재 (PDF)</h2>
              <p class="muted ellipsis" style="margin:4px 0 0;font-size:13px">${L.folder ? esc(L.folder) + ` · PDF ${L.total || 0}권 · ${(L.pages || 0).toLocaleString()}쪽${L.ocr ? ' · 글자 인식 ' + L.ocr + '쪽' : ''}` : '아직 폴더를 고르지 않았어요 · 스캔한 PDF 도 글자를 읽어 둬요'}</p></div>
            <button type="button" class="btn" data-act="libfolder">${L.folder ? '폴더 바꾸기' : '폴더 고르기'}</button>
            ${L.folder ? '<button type="button" class="btn" data-act="librescan">다시 읽기</button>' : ''}
          </section>

          <section class="card" aria-label="기본값">
            <h2 style="font-size:16px">기본값</h2>
            <label class="field">새 설교의 기본 장소 (섬기는 교회)<input class="input" id="homeplace" value="${esc((st.ui && st.ui.homePlace) || '')}" placeholder="예: 본 교회, ○○교회"></label>
          </section>
        </div>

        <aside class="aside">
          <section class="card" aria-label="화면 모드"><h2 style="font-size:16px">화면 모드</h2>
            <div class="seg">${[['system', '시스템 설정'], ['light', '라이트'], ['dark', '다크']].map(([k, l]) => `<button type="button" data-appearance="${k}" class="${(st.appearance || 'system') === k ? 'on' : ''}">${l}</button>`).join('')}</div></section>
          <section class="card" aria-label="성경 사이트"><h2 style="font-size:16px">성경 사이트</h2>
            <div class="tile row"><b class="grow">HOLYBIBLE</b><span class="muted" style="font-size:12px">biblenote.kr</span></div>
            <div class="faint" style="font-size:12px">본문 옆 ‘HOLYBIBLE에서 열기’를 누르면 그 장이 바로 열려요.</div></section>
          <section class="card" aria-label="업데이트">
            <div class="row"><h2 class="grow" style="font-size:16px">업데이트</h2><span class="chip sm" style="background:var(--ok-bg);color:var(--ok)">버전 ${esc(st.build || '')}${u.status === 'uptodate' ? ' · 최신' : ''}</span></div>
            <div class="muted" style="font-size:13px">${updateText(u)}</div>
            ${u.status === 'downloading' ? `<div class="progress"><i style="width:${Math.round((u.progress || 0) * 100)}%"></i></div>` : ''}
            <button type="button" class="toggle ${st.autoUpdate ? 'on' : ''}" data-act="auto" aria-pressed="${!!st.autoUpdate}"><span class="tr"><i></i></span>저절로 업데이트</button>
            <div class="row wrap">
              ${u.status === 'ready' ? '<button type="button" class="btn brand" data-act="restart">지금 다시 시작</button>' : u.status === 'available' ? '<button type="button" class="btn brand" data-act="install">새 버전 받기</button>' : '<button type="button" class="btn ink" data-act="check">업데이트 확인</button>'}
              ${u.canRollback ? `<button type="button" class="btn" data-act="rollback">이전 버전(${esc(u.backupVersion)})으로 되돌리기</button>` : ''}
            </div></section>
          <section class="card" aria-label="앱 정보">
            <div class="logo">Sermon<em>ROAD</em></div>
            <div class="muted" style="font-size:13px;line-height:1.6">개혁주의 설교자를 위한 설교 기록 · 준비 앱<br>기록은 모두 이 맥 안에 저장돼요</div>
            <div class="row wrap" style="gap:12px"><button type="button" class="link" data-act="diag">진단 정보 보기</button></div>
            <div class="made" style="text-align:left;padding:0">made by. Nevertheless_D</div>
          </section>
        </aside>
      </div>`;
    bind(el);
  }

  function updateText(u) {
    return ({
      checking: '새 버전이 있는지 보는 중…', uptodate: '최신 버전이에요. 새 버전이 나오면 앱이 알아서 받아요.',
      available: `새 버전 ${esc(u.latest && u.latest.version || '')}이 나왔어요.`, downloading: '새 버전을 받는 중…', installing: '새 버전을 설치하는 중…',
      ready: `새 버전${u.installed ? ' ' + esc(u.installed) : ''}이 준비됐어요. 다시 시작하면 바뀌어요.`, error: esc(u.error || '업데이트를 확인하지 못했어요'),
    })[u.status] || '새 버전이 나오면 앱이 알아서 받아요.';
  }

  async function saveSettings(patch) {
    try { S.settings = await api('/api/settings', patch); applyTheme(); } catch (e) { fail(e); }
  }

  function bind(el) {
    el.onclick = async (e) => {
      const t = e.target.closest('button');
      if (!t) return;
      const d = t.dataset;
      if (d.mode) { await saveSettings({ claudeMode: d.mode }); return render(el); }
      if (d.appearance) { await saveSettings({ appearance: d.appearance }); return render(el); }
      try {
        switch (d.act) {
          case 'savekey': {
            const k = $('#apikey', el).value.trim();
            if (!k) { toast('키를 붙여 넣어 주세요', { err: true }); return; }
            S.claude = await api('/api/claude/key', { key: k });
            toast('키를 저장했어요. 연결을 시험해 볼게요');
            render(el);
            try { await api('/api/claude/test', {}); toast('Claude 와 잘 연결됐어요'); } catch (err) { fail(err); }
            break;
          }
          case 'testkey': toast('연결을 시험하는 중…'); await api('/api/claude/test', {}); toast('Claude 와 잘 연결됐어요'); render(el); break;
          case 'delkey':
            if (await confirmBox('API 키를 지울까요?', '키체인에서 지워요. 다시 쓰려면 키를 새로 넣어야 해요.', '지우기', true)) { S.claude = await api('/api/claude/key', { key: '' }); render(el); }
            break;
          case 'howkey': howKey(); break;
          case 'libfolder': {
            const p = await chooseFolder('설교 서재(주석 · 강해 PDF)가 있는 폴더를 골라 주세요', S.lib && S.lib.folder);
            if (p) { S.lib = await api('/api/library/folder', { path: p }); toast('서재 폴더를 정했어요. PDF 를 읽기 시작해요'); render(el); }
            break;
          }
          case 'librescan': S.lib = await api('/api/library/rescan', {}); toast('서재를 다시 훑었어요'); render(el); break;
          case 'auto': await saveSettings({ autoUpdate: !S.settings.autoUpdate }); render(el); break;
          case 'check': S.update = await api('/api/update/check', {}); render(el); break;
          case 'install': await api('/api/update/install', {}); toast('새 버전을 받기 시작했어요'); break;
          case 'restart': await api('/api/update/restart', {}); break;
          case 'rollback':
            if (await confirmBox('이전 버전으로 되돌릴까요?', '되돌린 뒤에는 저절로 업데이트가 꺼져요. 기록은 그대로예요.', '되돌리기')) { await api('/api/update/rollback', {}); toast('되돌렸어요. 다시 시작하면 바뀌어요', { action: '다시 시작', onAction: () => api('/api/update/restart', {}) }); }
            break;
          case 'diag': openExternal('/diag'); break;
        }
      } catch (err) { fail(err); }
    };
    $('#showboth', el).onchange = (e) => saveSettings({ showBoth: e.target.checked });
    const model = $('#model', el); if (model) model.onchange = (e) => saveSettings({ model: e.target.value }).then(() => toast('모델을 바꿨어요'));
    $$('[data-std]', el).forEach((c) => c.onchange = () => saveSettings({ standards: $$('[data-std]', el).filter((x) => x.checked).map((x) => x.dataset.std) }));
    $('#extra', el).onchange = (e) => saveSettings({ extraPrompt: e.target.value }).then(() => toast('저장했어요'));
    $('#homeplace', el).onchange = (e) => saveSettings({ ui: { homePlace: e.target.value.trim() } }).then(() => toast('저장했어요'));
  }

  function howKey() {
    modal({
      title: 'Claude API 키 받는 방법', wide: true,
      body: `<ol style="margin:0;padding-left:20px;line-height:1.9;font-size:14px">
        <li><b>console.anthropic.com</b> 에 들어가 로그인해요 (지금 쓰시는 Claude 계정으로 들어갈 수 있어요).</li>
        <li>왼쪽 메뉴의 <b>Billing(결제)</b>에서 카드를 등록하고 사용 금액을 조금 충전해요. 설교 준비에 쓰는 정도는 보통 크지 않아요.</li>
        <li><b>API Keys</b> 메뉴에서 <b>Create Key</b> 를 눌러 이름(예: SermonROAD)을 정하고 키를 만들어요.</li>
        <li>나온 키(<b>sk-ant-</b> 로 시작)를 복사해서 이 화면의 API 키 칸에 붙여 넣고 <b>저장</b>을 눌러요.</li></ol>
        <div class="note"><b>알아 두세요</b> · 키는 이 맥의 키체인에만 저장되고 다른 곳으로 보내지 않아요. 키가 남에게 알려지면 콘솔에서 지우고 새로 만드세요. 지금 쓰시는 Claude 구독과 API 요금은 따로예요.</div>`,
      buttons: [{ label: '콘솔 열기', value: 'open' }, { label: '닫기', kind: 'brand', value: null }],
    }).then((v) => { if (v === 'open') openExternal('https://console.anthropic.com/settings/keys'); });
  }

  return { render: (el) => render(el), refresh: () => { const el = $('#main .page'); if (el && !el.contains(document.activeElement)) render(el); } };
})();
