/**
 * TechTracker Dashboard
 * Interactive rankings with sort, search, category filtering, and Chart.js trends.
 */

// ═══════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════
const state = {
  allRepos: [],
  groups: [],
  currentGroup: 'all',
  currentSort: { column: 'score', direction: 'desc' },
  historyCache: new Map(),
  chart: null,
  theme: 'dark',
};

// ═══════════════════════════════════════════════════════════
// Path resolution
// ═══════════════════════════════════════════════════════════
const isLocal = ['localhost', '127.0.0.1'].includes(window.location.hostname);
const BASE = isLocal ? '..' : '/TechTracker';

// Known category file names (fallback; groups derived from data)
const KNOWN_GROUPS = [
  'frontend_frameworks', 'backend_frameworks', 'mobile_frameworks',
  'testing_tools', 'devops_tools', 'databases', 'programming_languages',
  'design_tools',
];

// ═══════════════════════════════════════════════════════════
// URL State (hash-based deep linking)
// ═══════════════════════════════════════════════════════════
function readURLState() {
  const params = new URLSearchParams(window.location.hash.slice(1));
  if (params.has('group')) state.currentGroup = params.get('group');
  if (params.has('sort')) {
    state.currentSort.column = params.get('sort');
    state.currentSort.direction = params.get('dir') === 'asc' ? 'asc' : 'desc';
  }
  const q = params.get('q');
  if (q) {
    const input = document.getElementById('search-input');
    if (input) input.value = q;
  }
}

function writeURLState() {
  const params = new URLSearchParams();
  if (state.currentGroup !== 'all') params.set('group', state.currentGroup);
  if (state.currentSort.column !== 'score') params.set('sort', state.currentSort.column);
  if (state.currentSort.direction !== 'desc') params.set('dir', state.currentSort.direction);
  const q = document.getElementById('search-input')?.value?.trim();
  if (q) params.set('q', q);
  const hash = params.toString();
  const url = hash ? '#' + hash : window.location.pathname;
  history.replaceState(null, '', url);
}

// ═══════════════════════════════════════════════════════════
// Init
// ═══════════════════════════════════════════════════════════
document.addEventListener('DOMContentLoaded', async () => {
  initTheme();
  readURLState();
  renderSkeleton();
  setupEventListeners();

  const loaded = await loadData();

  if (loaded && state.allRepos.length > 0) {
    renderCategoryTabs();
    renderStats();
    restoreTabFromURL();
    refreshTable();
    await loadHistoryAndChart();
  } else if (!loaded) {
    renderErrorState();
  } else {
    renderEmptyState();
  }
});

// ═══════════════════════════════════════════════════════════
// Theme (CSS-driven icon visibility)
// ═══════════════════════════════════════════════════════════
function initTheme() {
  state.theme = document.documentElement.getAttribute('data-theme') || 'dark';

  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    if (!localStorage.getItem('techtracker-theme')) {
      state.theme = e.matches ? 'dark' : 'light';
      document.documentElement.setAttribute('data-theme', state.theme);
      updateChartTheme();
    }
  });
}

function toggleTheme() {
  state.theme = state.theme === 'light' ? 'dark' : 'light';
  localStorage.setItem('techtracker-theme', state.theme);
  document.documentElement.setAttribute('data-theme', state.theme);
  updateChartTheme();
}

// ═══════════════════════════════════════════════════════════
// Data Loading
// ═══════════════════════════════════════════════════════════
async function loadData() {
  try {
    const [allResp, metaResp] = await Promise.allSettled([
      fetch(`${BASE}/data/all.json`),
      fetch(`${BASE}/data/run-metadata.json`),
    ]);

    if (allResp.status !== 'fulfilled' || !allResp.value.ok) {
      console.error('Failed to load all.json');
      return false;
    }
    state.allRepos = await allResp.value.json();

    if (metaResp.status === 'fulfilled' && metaResp.value.ok) {
      const meta = await metaResp.value.json();
      const el = document.getElementById('last-updated');
      if (el) {
        el.textContent = `Updated ${formatRelative(meta.end_time)}`;
        el.title = formatDate(meta.end_time);
      }
    }

    // Load per-category JSON files in parallel
    const results = await Promise.allSettled(
      KNOWN_GROUPS.map(async (name) => {
        try {
          const resp = await fetch(`${BASE}/data/${name}.json`);
          if (!resp.ok) return null;
          const repos = await resp.json();
          return repos.length > 0 ? { name, repos } : null;
        } catch { return null; }
      })
    );

    state.groups = results
      .filter((r) => r.status === 'fulfilled' && r.value)
      .map((r) => ({
        name: r.value.name,
        label: r.value.name
          .replace(/_/g, ' ')
          .replace(/\b\w/g, (c) => c.toUpperCase()),
        repos: r.value.repos,
      }));
    return true;
  } catch (err) {
    console.error('Data load error:', err);
    return false;
  }
}

// ═══════════════════════════════════════════════════════════
// Skeleton / Error / Empty States
// ═══════════════════════════════════════════════════════════
function renderSkeleton() {
  const tbody = document.getElementById('table-body');
  if (!tbody) return;
  const rows = Array.from({ length: 8 }, () => `
    <tr>
      <td><div class="skeleton skeleton-text-short"></div></td>
      <td><div class="skeleton skeleton-text"></div></td>
      <td><div class="skeleton skeleton-text-short"></div></td>
      <td><div class="skeleton skeleton-text-short"></div></td>
      <td><div class="skeleton skeleton-text-short"></div></td>
      <td><div class="skeleton skeleton-tag"></div></td>
      <td><div class="skeleton skeleton-text-short"></div></td>
      <td><div class="skeleton skeleton-text-short"></div></td>
    </tr>
  `).join('');
  tbody.innerHTML = rows;
}

function renderEmptyState() {
  const tbody = document.getElementById('table-body');
  if (!tbody) return;
  tbody.innerHTML = `
    <tr>
      <td colspan="8">
        <div class="table-status">
          <div class="table-status-icon">📭</div>
          <div class="table-status-text">No data available</div>
          <div class="table-status-hint">Run <code>make run</code> to generate rankings, or check that <code>data/</code> files exist.</div>
        </div>
      </td>
    </tr>
  `;
}

function renderErrorState() {
  const tbody = document.getElementById('table-body');
  if (!tbody) return;
  tbody.innerHTML = `
    <tr>
      <td colspan="8">
        <div class="error-state">
          <div class="error-state-icon">⚠️</div>
          <div class="error-state-text">Failed to load data</div>
          <div class="error-state-hint">Check your connection or try again. If this persists, the data files may be missing.</div>
          <button class="error-state-retry" onclick="location.reload()">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
            Retry
          </button>
        </div>
      </td>
    </tr>
  `;
  // Also clear stats
  document.getElementById('stat-categories').textContent = '—';
  document.getElementById('stat-repos').textContent = '—';
  document.getElementById('stat-top-star').textContent = '—';
}

// ═══════════════════════════════════════════════════════════
// Category Tabs
// ═══════════════════════════════════════════════════════════
function renderCategoryTabs() {
  const container = document.getElementById('category-tabs');
  container.innerHTML = '<button class="tab active" role="tab" aria-selected="true" data-group="all">All Technologies</button>';

  state.groups.forEach((g) => {
    const btn = document.createElement('button');
    btn.className = 'tab';
    btn.setAttribute('role', 'tab');
    btn.setAttribute('aria-selected', 'false');
    btn.dataset.group = g.name;
    btn.textContent = `${g.label} (${g.repos.length})`;
    container.appendChild(btn);
  });
}

function restoreTabFromURL() {
  if (state.currentGroup === 'all') return;
  const tab = document.querySelector(`.tab[data-group="${state.currentGroup}"]`);
  if (tab) {
    document.querySelectorAll('.tab').forEach((t) => {
      t.classList.remove('active');
      t.setAttribute('aria-selected', 'false');
    });
    tab.classList.add('active');
    tab.setAttribute('aria-selected', 'true');
  } else {
    state.currentGroup = 'all';
  }
}

// ═══════════════════════════════════════════════════════════
// Stats Bar
// ═══════════════════════════════════════════════════════════
function renderStats() {
  document.getElementById('stat-categories').textContent = state.groups.length;
  document.getElementById('stat-repos').textContent = state.allRepos.length;

  const top = [...state.allRepos].sort((a, b) => (b.stargazers_count || 0) - (a.stargazers_count || 0))[0];
  const el = document.getElementById('stat-top-star');
  if (top) {
    el.textContent = top.name || top.full_name;
    el.title = `${(top.stargazers_count || 0).toLocaleString()} ★`;
    el.classList.add('is-text');
  }
}

// ═══════════════════════════════════════════════════════════
// Table Rendering
// ═══════════════════════════════════════════════════════════
function renderTable(repos) {
  const tbody = document.getElementById('table-body');
  const title = document.getElementById('table-title');

  const groupLabel = state.currentGroup === 'all'
    ? 'All Technologies'
    : state.groups.find((g) => g.name === state.currentGroup)?.label || 'Results';
  title.textContent = `${groupLabel} (${repos.length})`;

  if (repos.length === 0) {
    tbody.innerHTML = `
      <tr><td colspan="8">
        <div class="table-status">
          <div class="table-status-icon">🔍</div>
          <div class="table-status-text">No matches found</div>
          <div class="table-status-hint">Try a different search term or category.</div>
        </div>
      </td></tr>`;
    return;
  }

  const fragment = document.createDocumentFragment();
  repos.forEach((r, i) => {
    const tr = document.createElement('tr');
    tr.className = 'table-row row-entering';
    tr.dataset.fullname = r.full_name || '';
    tr.style.animationDelay = `${i * 20}ms`;

    const score = r.score != null ? r.score.toFixed(4) : '—';
    const lang = r.language || '';
    const name = r.name || r.full_name || '';
    const url = r.html_url || '';

    tr.innerHTML = `
      <td class="cell-rank">${i < 3 ? `<span class="rank-badge rank-${i + 1}">${i + 1}</span>` : `<span class="rank-normal">${i + 1}</span>`}</td>
      <td class="cell-name">${url ? `<a href="${esc(url)}" target="_blank" rel="noopener noreferrer">${esc(name)}</a>` : esc(name)}</td>
      <td class="cell-stars">${(r.stargazers_count || 0).toLocaleString()}</td>
      <td>${(r.forks_count || 0).toLocaleString()}</td>
      <td>${(r.open_issues_count || 0).toLocaleString()}</td>
      <td>${lang ? `<span class="language-tag">${esc(lang)}</span>` : '<span class="language-tag">—</span>'}</td>
      <td class="cell-score">${score}</td>
      <td>${buildTrend(r)}</td>
    `;

    fragment.appendChild(tr);
  });

  tbody.innerHTML = '';
  tbody.appendChild(fragment);
}

function buildTrend(repo) {
  const parts = [];

  if (repo.star_delta && repo.star_delta !== 0) {
    const cls = repo.star_delta > 0 ? 'trend-up' : 'trend-down';
    const arrow = repo.star_delta > 0 ? '↑' : '↓';
    const sign = repo.star_delta > 0 ? '+' : '';
    parts.push(`<span class="trend ${cls}">${arrow} ${sign}${repo.star_delta.toLocaleString()} ★</span>`);
  }

  if (repo.rank_change && repo.rank_change !== 0) {
    const cls = repo.rank_change > 0 ? 'trend-up' : 'trend-down';
    const arrow = repo.rank_change > 0 ? '▲' : '▼';
    const sign = repo.rank_change > 0 ? '+' : '';
    parts.push(`<span class="trend ${cls}">${arrow} ${sign}${repo.rank_change}</span>`);
  }

  return parts.length > 0 ? parts.join(' ') : '<span class="trend trend-neutral">━</span>';
}

// ═══════════════════════════════════════════════════════════
// Sorting
// ═══════════════════════════════════════════════════════════
const SORT_KEYS = {
  rank: (r, i) => i,
  name: (r) => (r.name || r.full_name || '').toLowerCase(),
  stars: (r) => r.stargazers_count || 0,
  forks: (r) => r.forks_count || 0,
  issues: (r) => r.open_issues_count || 0,
  language: (r) => (r.language || '').toLowerCase(),
  score: (r) => r.score || 0,
};

function sortRepos(repos) {
  const { column, direction } = state.currentSort;
  const keyFn = SORT_KEYS[column] || SORT_KEYS.score;
  const indexed = repos.map((r, i) => [r, i]);
  indexed.sort((a, b) => {
    const va = keyFn(a[0], a[1]);
    const vb = keyFn(b[0], b[1]);
    if (typeof va === 'string') return va.localeCompare(vb);
    return va - vb;
  });
  const sorted = indexed.map((x) => x[0]);
  return direction === 'desc' ? sorted.reverse() : sorted;
}

function updateSortHeaders() {
  document.querySelectorAll('th.sortable').forEach((th) => {
    th.classList.remove('active', 'asc', 'desc');
    th.removeAttribute('aria-sort');
    if (th.dataset.sort === state.currentSort.column) {
      th.classList.add('active', state.currentSort.direction);
      th.setAttribute('aria-sort', state.currentSort.direction === 'asc' ? 'ascending' : 'descending');
    }
  });
}

// ═══════════════════════════════════════════════════════════
// Filtering
// ═══════════════════════════════════════════════════════════
function filterBySearch(repos, query) {
  if (!query) return repos;
  const q = query.toLowerCase();
  return repos.filter((r) =>
    (r.name || '').toLowerCase().includes(q) ||
    (r.full_name || '').toLowerCase().includes(q) ||
    (r.description || '').toLowerCase().includes(q) ||
    (r.language || '').toLowerCase().includes(q)
  );
}

function getCurrentRepos() {
  if (state.currentGroup === 'all') return [...state.allRepos];
  const g = state.groups.find((g) => g.name === state.currentGroup);
  return g ? [...g.repos] : [];
}

function refreshTable() {
  let repos = getCurrentRepos();
  const query = document.getElementById('search-input')?.value || '';
  repos = filterBySearch(repos, query);
  repos = sortRepos(repos);
  renderTable(repos);
  updateSortHeaders();
  writeURLState();
  updateChartBadge();
}

// ═══════════════════════════════════════════════════════════
// History & Chart
// ═══════════════════════════════════════════════════════════
async function fetchHistoryDay(dateStr) {
  try {
    const resp = await fetch(`${BASE}/data/history/${dateStr}.json`);
    if (resp.ok) {
      const data = await resp.json();
      state.historyCache.set(dateStr, data);
      return { date: dateStr, repos: data };
    }
  } catch { /* day may not exist */ }
  return null;
}

async function loadHistoryAndChart() {
  const chartNote = document.getElementById('chart-note');

  // Build date list for last 30 days
  const dates = [];
  for (let i = 1; i <= 30; i++) {
    const d = new Date();
    d.setDate(d.getDate() - i);
    dates.push(d.toISOString().split('T')[0]);
  }

  // Fetch all in parallel (limited concurrency)
  const CONCURRENCY = 6;
  const results = [];
  for (let i = 0; i < dates.length; i += CONCURRENCY) {
    const batch = dates.slice(i, i + CONCURRENCY).map(fetchHistoryDay);
    const batchResults = await Promise.all(batch);
    results.push(...batchResults);
  }

  const historyData = results.filter(Boolean);

  if (historyData.length < 2) {
    chartNote.innerHTML = 'Need at least 2 days of history for trend charts. Run <code>make run</code> daily to build data.';
    return;
  }

  chartNote.innerHTML = `Showing score trends over <strong>${historyData.length} days</strong> of data.`;
  renderTrendChart(historyData);
}

function updateChartBadge() {
  const badge = document.getElementById('chart-badge');
  if (!badge) return;
  badge.textContent = state.currentGroup === 'all' ? 'Top 5' : 'Top 5 in Category';
}

function getTopForChart() {
  const repos = getCurrentRepos();
  const sorted = sortRepos(repos);
  return sorted.slice(0, 5);
}

function renderTrendChart(historyData) {
  const ctx = document.getElementById('trend-chart').getContext('2d');
  const top5 = getTopForChart();

  const colors = [
    getComputedStyle(document.documentElement).getPropertyValue('--chart-1').trim() || '#4f46e5',
    getComputedStyle(document.documentElement).getPropertyValue('--chart-2').trim() || '#7c3aed',
    getComputedStyle(document.documentElement).getPropertyValue('--chart-3').trim() || '#059669',
    getComputedStyle(document.documentElement).getPropertyValue('--chart-4').trim() || '#d97706',
    getComputedStyle(document.documentElement).getPropertyValue('--chart-5').trim() || '#e11d48',
  ];

  const sortedHistory = [...historyData].sort((a, b) => a.date.localeCompare(b.date));

  const datasets = top5.map((repo, i) => {
    const scores = sortedHistory.map((day) => {
      const found = day.repos.find((r) => r.full_name === repo.full_name);
      return found ? found.score : null;
    });

    // Fill gaps: carry forward last known value
    let lastKnown = null;
    for (let j = 0; j < scores.length; j++) {
      if (scores[j] != null) lastKnown = scores[j];
      else scores[j] = lastKnown;
    }

    return {
      label: repo.name || repo.full_name,
      data: scores,
      borderColor: colors[i % colors.length],
      backgroundColor: colors[i % colors.length] + '20',
      borderWidth: 2.5,
      tension: 0.35,
      fill: false,
      pointRadius: 3,
      pointHoverRadius: 7,
      pointBackgroundColor: colors[i % colors.length],
    };
  }).filter((ds) => ds.data.some((v) => v != null));

  if (state.chart) state.chart.destroy();

  const isDark = state.theme === 'dark';
  const gridColor = isDark ? 'rgba(148, 163, 184, 0.1)' : 'rgba(0, 0, 0, 0.06)';
  const textColor = isDark ? '#94a3b8' : '#64748b';

  state.chart = new Chart(ctx, {
    type: 'line',
    data: {
      labels: sortedHistory.map((d) => {
        const date = new Date(d.date + 'T00:00:00');
        return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
      }),
      datasets,
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: {
        mode: 'index',
        intersect: false,
      },
      plugins: {
        legend: {
          position: 'bottom',
          labels: {
            usePointStyle: true,
            padding: 24,
            color: textColor,
            font: { size: 13, family: "'Inter', sans-serif" },
          },
        },
        tooltip: {
          backgroundColor: isDark ? '#1e293b' : '#ffffff',
          titleColor: textColor,
          bodyColor: isDark ? '#f1f5f9' : '#0f172a',
          borderColor: isDark ? '#334155' : '#e2e8f0',
          borderWidth: 1,
          padding: 12,
          cornerRadius: 8,
          callbacks: {
            label: (ctx) => `  ${ctx.dataset.label}: ${ctx.parsed.y.toFixed(4)}`,
          },
        },
      },
      scales: {
        y: {
          min: 0,
          max: 1,
          ticks: {
            callback: (v) => v.toFixed(2),
            color: textColor,
            font: { size: 11 },
          },
          grid: { color: gridColor },
          title: {
            display: true,
            text: 'Popularity Score',
            color: textColor,
            font: { size: 12, weight: '600' },
          },
        },
        x: {
          ticks: { color: textColor, font: { size: 11 } },
          grid: { color: gridColor },
          title: {
            display: true,
            text: 'Date',
            color: textColor,
            font: { size: 12, weight: '600' },
          },
        },
      },
    },
  });
}

function updateChartTheme() {
  if (!state.chart) return;
  const isDark = state.theme === 'dark';
  const textColor = isDark ? '#94a3b8' : '#64748b';
  const gridColor = isDark ? 'rgba(148, 163, 184, 0.1)' : 'rgba(0, 0, 0, 0.06)';

  state.chart.options.plugins.legend.labels.color = textColor;
  state.chart.options.plugins.tooltip.backgroundColor = isDark ? '#1e293b' : '#ffffff';
  state.chart.options.plugins.tooltip.titleColor = textColor;
  state.chart.options.plugins.tooltip.bodyColor = isDark ? '#f1f5f9' : '#0f172a';
  state.chart.options.plugins.tooltip.borderColor = isDark ? '#334155' : '#e2e8f0';
  state.chart.options.scales.y.ticks.color = textColor;
  state.chart.options.scales.y.grid.color = gridColor;
  state.chart.options.scales.y.title.color = textColor;
  state.chart.options.scales.x.ticks.color = textColor;
  state.chart.options.scales.x.grid.color = gridColor;
  state.chart.options.scales.x.title.color = textColor;
  state.chart.update('none');
}

// ═══════════════════════════════════════════════════════════
// Chart refresh on category change
// ═══════════════════════════════════════════════════════════
async function refreshChart() {
  if (state.chart && state.historyCache.size > 1) {
    // Re-render with current category's top 5
    const historyData = Array.from(state.historyCache.entries())
      .map(([date, repos]) => ({ date, repos }))
      .sort((a, b) => a.date.localeCompare(b.date));
    if (historyData.length >= 2) renderTrendChart(historyData);
  }
}

// ═══════════════════════════════════════════════════════════
// Event Listeners
// ═══════════════════════════════════════════════════════════
function setupEventListeners() {
  // Theme toggle
  document.getElementById('theme-toggle')?.addEventListener('click', toggleTheme);

  // Category tabs
  document.getElementById('category-tabs')?.addEventListener('click', (e) => {
    const tab = e.target.closest('.tab');
    if (!tab) return;

    document.querySelectorAll('.tab').forEach((t) => {
      t.classList.remove('active');
      t.setAttribute('aria-selected', 'false');
    });
    tab.classList.add('active');
    tab.setAttribute('aria-selected', 'true');

    state.currentGroup = tab.dataset.group;
    state.currentSort = { column: 'score', direction: 'desc' };
    refreshTable();
    refreshChart();
  });

  // Column sorting
  document.querySelector('#ranking-table thead')?.addEventListener('click', (e) => {
    const th = e.target.closest('th.sortable');
    if (!th) return;
    const column = th.dataset.sort;

    if (state.currentSort.column === column) {
      state.currentSort.direction = state.currentSort.direction === 'asc' ? 'desc' : 'asc';
    } else {
      state.currentSort.column = column;
      state.currentSort.direction = 'desc';
    }
    refreshTable();
  });

  // Search with debounce
  let searchTimer;
  document.getElementById('search-input')?.addEventListener('input', (e) => {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(() => refreshTable(), 150);
  });

  // Row selection
  document.getElementById('table-body')?.addEventListener('click', (e) => {
    const row = e.target.closest('.table-row');
    if (!row) return;
    document.querySelectorAll('.table-row').forEach((r) => r.classList.remove('selected'));
    row.classList.add('selected');
  });

  // Back to top — smart threshold based on viewport height
  const backBtn = document.getElementById('back-to-top');
  window.addEventListener('scroll', () => {
    if (!backBtn) return;
    backBtn.classList.toggle('visible', window.scrollY > window.innerHeight * 0.75);
  });
  backBtn?.addEventListener('click', () => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  });

  // Keyboard shortcut: / to focus search
  document.addEventListener('keydown', (e) => {
    if (e.key === '/' && document.activeElement?.tagName !== 'INPUT' && document.activeElement?.tagName !== 'TEXTAREA') {
      e.preventDefault();
      document.getElementById('search-input')?.focus();
    }
  });
}

// ═══════════════════════════════════════════════════════════
// Utilities
// ═══════════════════════════════════════════════════════════
function esc(str) {
  if (!str) return '';
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

function formatDate(isoString) {
  if (!isoString) return '';
  try {
    const d = new Date(isoString);
    return d.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return isoString;
  }
}

function formatRelative(isoString) {
  if (!isoString) return '';
  try {
    const then = new Date(isoString);
    const now = new Date();
    const diffMs = now - then;
    const diffMin = Math.floor(diffMs / 60000);
    const diffHrs = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMin < 1) return 'just now';
    if (diffMin < 60) return `${diffMin}m ago`;
    if (diffHrs < 24) return `${diffHrs}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return formatDate(isoString);
  } catch {
    return isoString;
  }
}
