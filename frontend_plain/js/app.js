// ===== Конфиг API =====
// Измени это на адрес своего бэкенда (Gin)
const DEFAULT_API_BASE = 'http://127.0.0.1:8080';

// ===== Селекторы =====
const apiBaseInput = document.getElementById('apiBase');
const saveApiBaseBtn = document.getElementById('saveApiBase');
const apiStatus = document.getElementById('apiStatus');

const longLinkInput = document.getElementById('longLink');
const customCodeInput = document.getElementById('customCode');
const createBtn = document.getElementById('createBtn');
const copyBtn = document.getElementById('copyBtn');
const createError = document.getElementById('createError');
const resultBox = document.getElementById('resultBox');
const shortLinkA = document.getElementById('shortLink');
const longPreview = document.getElementById('longPreview');

const anCodeInput = document.getElementById('anCode');
const loadAnBtn = document.getElementById('loadAnBtn');
const anError = document.getElementById('anError');
const anBox = document.getElementById('anBox');
const totalCountSpan = document.getElementById('totalCount');
const anTableBody = document.querySelector('#anTable tbody');

// ===== Хранилище настроек =====
function getApiBase() {
  return localStorage.getItem('apiBase') || DEFAULT_API_BASE;
}
function setApiBase(v) {
  localStorage.setItem('apiBase', v);
}

// Инициализация
apiBaseInput.value = getApiBase();

saveApiBaseBtn.addEventListener('click', () => {
  setApiBase(apiBaseInput.value.trim().replace(/\/$/,''));
  apiStatus.textContent = 'Сохранено ✓';
  setTimeout(() => apiStatus.textContent = '', 1500);
});

// ===== Утилиты =====
async function api(url, options = {}) {
  const base = getApiBase();
  const res = await fetch(base.replace(/\/$/,'') + url, {
    headers: { 'Content-Type': 'application/json', ...(options.headers||{}) },
    ...options,
  });
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch (_) {}
  if (!res.ok) {
    const message = (data && data.message) || text || ('HTTP ' + res.status);
    throw new Error(message);
  }
  return data;
}
function isValidUrl(str) {
  return /^https?:\/\//i.test(str);
}
function isValidCode(str) {
  return /^[A-Za-z0-9_-]{6}$/.test(str);
}
function fmtDate(s) {
  const d = new Date(s);
  return d.toLocaleString();
}

// ===== Сокращение ссылки =====
createBtn.addEventListener('click', async () => {
  createError.textContent = '';
  resultBox.classList.add('hidden');
  shortLinkA.textContent = '';
  longPreview.textContent = '';

  const longLink = longLinkInput.value.trim();
  const custom = customCodeInput.value.trim();

  try {
    if (!isValidUrl(longLink)) throw new Error('Введите корректный URL (например, https://example.com)');
    if (custom && !isValidCode(custom)) throw new Error("Код должен быть ровно 6 символов: A–Z, a–z, 0–9, '-', '_'");    

    createBtn.disabled = true;
    const payload = { link: longLink, custom };
    const data = await api('/shorten', { method: 'POST', body: JSON.stringify(payload) });

    // Ожидается ответ вида { short: string, long: string }
    if (!data || !data.short) throw new Error('Неожиданный ответ бэкенда');
    shortLinkA.href = data.short;
    shortLinkA.textContent = data.short;
    longPreview.textContent = data.long || longLink;
    resultBox.classList.remove('hidden');
  } catch (e) {
    createError.textContent = e.message || 'Ошибка при создании ссылки';
  } finally {
    createBtn.disabled = false;
  }
});

copyBtn.addEventListener('click', async () => {
  const text = shortLinkA.textContent.trim();
  if (!text) return;
  try { await navigator.clipboard.writeText(text); } catch(_){}
});

// ===== Аналитика =====
loadAnBtn.addEventListener('click', async () => {
  anError.textContent = '';
  anBox.classList.add('hidden');
  anTableBody.innerHTML = '';
  totalCountSpan.textContent = '0';

  const code = anCodeInput.value.trim();
  try {
    if (!isValidCode(code)) throw new Error("Код должен быть 6 символов: A–Z, a–z, 0–9, '-', '_'");    

    loadAnBtn.disabled = true;
    const data = await api('/analytics/' + encodeURIComponent(code), { method: 'GET' });

    // Ожидается { redirects: Redirect[], total_count: number }
    const rows = (data && data.redirects) || [];
    totalCountSpan.textContent = String(data?.total_count ?? rows.length);

    rows.forEach((r, idx) => {
      const tr = document.createElement('tr');
      tr.innerHTML = [
        `<td>${idx + 1}</td>`,
        `<td>${r.created_at ? fmtDate(r.created_at) : ''}</td>`,
        `<td class="break">${r.user_agent || ''}</td>`,
        `<td class="break">${r.short_link || ''}</td>`,
        `<td class="break"><a href="${r.long_link || '#'}" target="_blank" rel="noreferrer">${r.long_link || ''}</a></td>`,
      ].join('');
      anTableBody.appendChild(tr);
    });

    anBox.classList.remove('hidden');
  } catch (e) {
    anError.textContent = e.message || 'Ошибка аналитики';
  } finally {
    loadAnBtn.disabled = false;
  }
});
