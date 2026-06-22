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
  theme: 'light',
};

// ═══════════════════════════════════════════════════════════
// Path resolution
// ═══════════════════════════════════════════════════════════
const isLocal = ['localhost', '127.0.0.1'].includes(window.location.hostname);
const BASE = isLocal ? '..' : '/TechTracker';

// ═══════════════════════════════════════════════════════════
// Init
// ═══════════════════════════════════════════════════════════
document.addEventListener('DOMContentLoaded', async () => {
  initTheme();
  renderSkeleton();
  setupEventListeners();

  await loadData();

  if (state.allRepos.length > 0) {
    renderCategoryTabs();
    renderStats();
    renderTable(state.allRepos);
    await loadHistoryAndChart();
  } else {
    renderEmptyState();
  }
});

// ═══════════════════════════════════════════════════════════
// Theme
// ═══════════════════════════════════════════════════════════
function initTheme() {
  const saved = localStorage.getItem('techtracker-theme');
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  state.theme = saved || (prefersDark ? 'dark' : 'light');
  applyTheme();

  // Listen for OS changes
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    if (!localStorage.getItem('techtracker-theme')) {
      state.theme = e.matches ? 'dark' : 'light';
      applyTheme();
    }
  });
}

function toggleTheme() {
  state.theme = state.theme === 'light' ? 'dark' : 'light';
  localStorage.setItem('techtracker-theme', state.theme);
  applyTheme();
}

function applyTheme() {
  document.documentElement.setAttribute('data-theme', state.theme);
  const icon = document.querySelector('.theme-icon-light');
  const iconDark = document.querySelector('.theme-icon-dark');
  if (icon && iconDark) {
    icon.style.display = state.theme === 'light' ? 'none' : 'inline';
    iconDark.style.display = state.theme === 'dark' ? 'none' : 'inline';
  }
  if (state.chart) {
    updateChartTheme();
  }
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

    if (allResp.status === 'fulfilled' && allResp.value.ok) {
      state.allRepos = await allResp.value.json();
    } else {
      console.error('Failed to load all.json');
      return;
    }

    if (metaResp.status === 'fulfilled' && metaResp.value.ok) {
      const meta = await metaResp.value.json();
      const el = document.getElementById('last-updated');
      if (el) el.textContent = `Updated ${formatDate(meta.end_time)}`;
    }

    // Load per-category JSON files
    const groupFiles = [
      'frontend_frameworks', 'backend_frameworks', 'mobile_frameworks',
      'testing_tools', 'devops_tools', 'databases', 'programming_languages',
      'design_tools',
    ];

    const results = await Promise.allSettled(
      groupFiles.map(async (name) => {
        const resp = await fetch(`${BASE}/data/${name}.json`);
        if (!resp.ok) return null;
        const repos = await resp.json();
        return repos.length > 0 ? { name, repos } : null;
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
  } catch (err) {
    console.error('Data load error:', err);
  }
}

// ═══════════════════════════════════════════════════════════
// Skeleton Loading
// ═══════════════════════════════════════════════════════════
function renderSkeleton() {
  const tbody = document.getElementById('table-body');
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

// ═══════════════════════════════════════════════════════════
// Stats Bar
// ═══════════════════════════════════════════════════════════
function renderStats() {
  document.getElementById('stat-categories').textContent = state.groups.length;
  document.getElementById('stat-repos').textContent = state.allRepos.length;

  const top = [...state.allRepos].sort((a, b) => (b.stargazers_count || 0) - (a.stargazers_count || 0))[0];
  if (top) {
    document.getElementById('stat-top-star').textContent = top.name || top.full_name;
    document.getElementById('stat-top-star').title = `${(top.stargazers_count || 0).toLocaleString()} ★`;
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
    tr.className = 'table-row';
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
  const sorted = [...repos].sort((a, b) => {
    const va = keyFn(a, repos.indexOf(a));
    const vb = keyFn(b, repos.indexOf(b));
    if (typeof va === 'string') return va.localeCompare(vb);
    return va - vb;
  });
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
}

// ═══════════════════════════════════════════════════════════
// History & Chart
// ═══════════════════════════════════════════════════════════
async function loadHistoryAndChart() {
  const chartNote = document.getElementById('chart-note');
  const historyData = [];

  for (let i = 1; i <= 30; i++) {
    const d = new Date();
    d.setDate(d.getDate() - i);
    const dateStr = d.toISOString().split('T')[0];
    try {
      const resp = await fetch(`${BASE}/data/history/${dateStr}.json`);
      if (resp.ok) {
        const data = await resp.json();
        state.historyCache.set(dateStr, data);
        historyData.push({ date: dateStr, repos: data });
      }
    } catch { /* day may not exist */ }
  }

  if (historyData.length < 2) {
    chartNote.innerHTML = 'Need at least 2 days of history for trend charts. Run <code>make run</code> daily to build data.';
    return;
  }

  chartNote.innerHTML = `Showing score trends over <strong>${historyData.length} days</strong> of data.`;
  renderTrendChart(historyData);
}

function renderTrendChart(historyData) {
  const ctx = document.getElementById('trend-chart').getContext('2d');
  const top5 = state.allRepos.slice(0, 5);

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

  // Back to top
  const backBtn = document.getElementById('back-to-top');
  window.addEventListener('scroll', () => {
    if (!backBtn) return;
    backBtn.classList.toggle('visible', window.scrollY > 600);
  });
  backBtn?.addEventListener('click', () => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  });

  // Keyboard shortcut: / to focus search
  document.addEventListener('keydown', (e) => {
    if (e.key === '/' && document.activeElement?.tagName !== 'INPUT') {
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
