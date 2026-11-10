// SermonROAD — Windows 판 화면 맞춤 (맥에서는 아무것도 하지 않아요)
//   · 화면 문구의 맥 말(키체인 · Finder · ⌘ · '이 맥')을 Windows 말로 바꿔요
//   · 폴더 고르기: 엔진이 Windows 폴더 선택 창을 띄워요 (/api/choose-folder)
// made by. Nevertheless_D
(function () {
  const isWin = /Windows/i.test(navigator.userAgent) || /^win/i.test(navigator.platform || '');
  if (!isWin) return;
  document.documentElement.dataset.os = 'windows';

  // 순서가 중요해요: 긴 말부터 바꿔요. ('맥추감사' 같은 말은 건드리지 않아요)
  const WORDS = [
    ['이 맥의 키체인', '이 컴퓨터의 Windows 자격 증명 관리자'],
    ['키체인에서', 'Windows 자격 증명 관리자에서'],
    ['키체인에', 'Windows 자격 증명 관리자에'],
    ['키체인', 'Windows 자격 증명 관리자'],
    ['이 맥 안에만', '이 컴퓨터 안에만'],
    ['이 맥 안에', '이 컴퓨터 안에'],
    ['이 맥 안', '이 컴퓨터 안'],
    ['이 맥의', '이 컴퓨터의'],
    ['이 맥에서', '이 컴퓨터에서'],
    ['이 맥', '이 컴퓨터'],
    ['맥 안에서', '컴퓨터 안에서'],
    ['Finder 에서', '파일 탐색기에서'],
    ['Finder에서', '파일 탐색기에서'],
    ['Finder', '파일 탐색기'],
    ['⌘', 'Ctrl+'],
  ];
  const fix = (s) => {
    if (!s) return s;
    let out = s;
    for (const [a, b] of WORDS) if (out.includes(a)) out = out.split(a).join(b);
    return out;
  };
  const fixNode = (root) => {
    if (root.nodeType === 3) { const t = fix(root.nodeValue); if (t !== root.nodeValue) root.nodeValue = t; return; }
    if (root.nodeType !== 1) return;
    for (const at of ['title', 'placeholder', 'aria-label']) {
      const v = root.getAttribute && root.getAttribute(at);
      if (v) { const t = fix(v); if (t !== v) root.setAttribute(at, t); }
    }
    const w = document.createTreeWalker(root, NodeFilter.SHOW_TEXT | NodeFilter.SHOW_ELEMENT);
    let n;
    while ((n = w.nextNode())) {
      if (n.nodeType === 3) { const t = fix(n.nodeValue); if (t !== n.nodeValue) n.nodeValue = t; }
      else if (n.nodeName === 'INPUT' || n.nodeName === 'TEXTAREA') {
        const v = n.getAttribute('placeholder'); if (v) { const t = fix(v); if (t !== v) n.setAttribute('placeholder', t); }
      } else {
        for (const at of ['title', 'aria-label']) {
          const v = n.getAttribute(at); if (v) { const t = fix(v); if (t !== v) n.setAttribute(at, t); }
        }
      }
    }
  };
  const start = () => {
    fixNode(document.body);
    new MutationObserver((ms) => {
      for (const m of ms) {
        if (m.type === 'characterData') { const t = fix(m.target.nodeValue); if (t !== m.target.nodeValue) m.target.nodeValue = t; }
        else for (const n of m.addedNodes) {
          // 사용자가 쓰는 글(입력 칸 · 원고)은 건드리지 않아요
          if (n.nodeType === 3 && n.parentNode && /^(TEXTAREA|INPUT)$/.test(n.parentNode.nodeName)) continue;
          if (n.nodeType === 1 && n.isContentEditable) continue;
          fixNode(n);
        }
      }
    }).observe(document.body, { childList: true, subtree: true, characterData: true });
  };
  if (document.body) start(); else document.addEventListener('DOMContentLoaded', start);

  // 폴더 고르기: Windows 의 폴더 선택 창
  window.chooseFolder = async function (prompt, startDir) {
    try {
      const r = await api('/api/choose-folder', { prompt, start: startDir || '' });
      return r && r.path ? r.path : '';
    } catch (e) {
      toast((e && e.message) || '폴더 고르기 창을 열지 못했어요');
      return '';
    }
  };
})();
