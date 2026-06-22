/**
 * TechTracker Dashboard
 * Progressive loading: metadata → chunked data + lazy groups + top5 chart history.
 */

// ═══════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════
const state = {
  // Bootstrap
  latestDate: '',
  categoryMeta: [],

  // Data
  allRepos: [],           // accumulated from chunks
  loadedChunks: 0,
  allChunksTotal: 0,
  groups: [],             // { name, label, repos } — populated on demand
  groupCache: {},         // key → repos (lazy loaded)
  currentGroup: 'all',
  currentSort: { column: 'score', direction: 'desc' },
  isLoadingChunk: false,

  // History & chart
  historyRange: 30,
  availableDates: [],
  top5History: null,
  chart: null,
  theme: 'dark',
};

// ═══════════════════════════════════════════════════════════
// Path resolution
// ═══════════════════════════════════════════════════════════
const isLocal = ['localhost', '127.0.0.1'].includes(window.location.hostname);
const BASE = isLocal ? '..' : '/TechTracker';

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
  setupScrollObserver();

  const loaded = await loadMetadata();
  if (!loaded) {
    renderErrorState();
    return;
  }

  renderStatsFromMeta();
  renderCategoryTabs();
  restoreTabFromURL();

  // Start loading data for the active tab.
  if (state.currentGroup === 'all') {
    await loadNextChunk();
  } else {
    await switchToGroup(state.currentGroup);
  }

  // Chart: load lightweight top5 history.
  await loadChartFromTop5History();
});

// ═══════════════════════════════════════════════════════════
// Theme
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
// Metadata Loading (fast, ~1KB)
// ═══════════════════════════════════════════════════════════
async function loadMetadata() {
  try {
    const resp = await fetch(`${BASE}/data/metadata.json`);
    if (!resp.ok) {
      console.error('Failed to load metadata.json');
      return false;
    }
    const meta = await resp.json();

    state.latestDate = meta.latest_date || '';
    state.allChunksTotal = meta.all_chunks || 0;
    state.categoryMeta = meta.categories || [];
    state.availableDates = meta.history?.available_dates || [];

    // Update last-updated display.
    const el = document.getElementById('last-updated');
    if (el && meta.end_time) {
      el.textContent = `Updated ${formatRelative(meta.end_time)}`;
      el.title = formatDate(meta.end_time);
    }

    return true;
  } catch (err) {
    console.error('Metadata load error:', err);
    return false;
  }
}

// ═══════════════════════════════════════════════════════════
// Chunk Loading (infinite scroll)
// ═══════════════════════════════════════════════════════════
async function loadNextChunk() {
  if (state.isLoadingChunk) return;
  if (state.loadedChunks >= state.allChunksTotal) return;

  state.isLoadingChunk = true;
  showChunkLoader(true);

  const n = state.loadedChunks + 1;
  try {
    const resp = await fetch(`${BASE}/data/${state.latestDate}/all/chunk_${n}.json`);
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const repos = await resp.json();

    state.allRepos.push(...repos);
    state.loadedChunks = n;

    refreshTable();
  } catch (err) {
    console.error(`Chunk ${n} load failed:`, err);
  } finally {
    state.isLoadingChunk = false;
    showChunkLoader(false);
  }
}

function setupScrollObserver() {
  const sentinel = document.getElementById('chunk-sentinel');
  if (!sentinel) return;

  const observer = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting && state.currentGroup === 'all') {
      loadNextChunk();
    }
  }, { rootMargin: '200px' });

  observer.observe(sentinel);
}

function showChunkLoader(show) {
  const el = document.getElementById('chunk-loader');
  if (el) el.style.display = show ? 'flex' : 'none';
}

// ═══════════════════════════════════════════════════════════
// Group (Category) Loading — lazy, on tab click
// ═══════════════════════════════════════════════════════════
async function loadGroupData(key) {
  if (state.groupCache[key]) return state.groupCache[key];

  try {
    const resp = await fetch(`${BASE}/data/${state.latestDate}/groups/${key}.json`);
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const repos = await resp.json();
    state.groupCache[key] = repos;
    return repos;
  } catch (err) {
    console.error(`Group ${key} load failed:`, err);
    return [];
  }
}

async function switchToGroup(key) {
  state.currentGroup = key;
  state.currentSort = { column: 'score', direction: 'desc' };

  if (key === 'all') {
    // Already loaded via chunks; just refresh.
  } else {
    const repos = await loadGroupData(key);
    // Ensure groups array has this entry.
    const existing = state.groups.find((g) => g.name === key);
    const meta = state.categoryMeta.find((c) => c.key === key);
    if (existing) {
      existing.repos = repos;
    } else {
      state.groups.push({
        name: key,
        label: meta?.label || key,
        repos,
      });
    }
  }

  refreshTable();
  refreshChart();
}

// ═══════════════════════════════════════════════════════════
// Skeleton / Error / Empty States
// ═══════════════════════════════════════════════════════════
function renderSkeleton() {
  const tbody = document.getElementById('table-body');
  if (!tbody) return;
  tbody.innerHTML = Array.from({ length: 8 }, () => `
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
}

function renderErrorState() {
  const tbody = document.getElementById('table-body');
  if (!tbody) return;
  tbody.innerHTML = `
    <tr><td colspan="8">
      <div class="error-state">
        <div class="error-state-icon">⚠️</div>
        <div class="error-state-text">Failed to load data</div>
        <div class="error-state-hint">Check your connection or try again.</div>
        <button class="error-state-retry" onclick="location.reload()">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="23 4 23 10 17 10"/><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/></svg>
          Retry
        </button>
      </div>
    </td></tr>`;
  ['stat-categories', 'stat-repos', 'stat-top-star'].forEach((id) => {
    document.getElementById(id).textContent = '—';
  });
}

// ═══════════════════════════════════════════════════════════
// Category Tabs
// ═══════════════════════════════════════════════════════════
function renderCategoryTabs() {
  const container = document.getElementById('category-tabs');
  container.innerHTML = '<button class="tab active" role="tab" aria-selected="true" data-group="all">All Technologies</button>';

  state.categoryMeta.forEach((c) => {
    const btn = document.createElement('button');
    btn.className = 'tab';
    btn.setAttribute('role', 'tab');
    btn.setAttribute('aria-selected', 'false');
    btn.dataset.group = c.key;
    btn.textContent = `${c.label} (${c.count})`;
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
// Stats Bar (from metadata)
// ═══════════════════════════════════════════════════════════
function renderStatsFromMeta() {
  document.getElementById('stat-categories').textContent = state.categoryMeta.length;
  document.getElementById('stat-repos').textContent = state.allChunksTotal > 0
    ? (state.allChunksTotal * 50 > 177 ? 177 : state.allChunksTotal * 50) // approximate
    : '...';

  // We'll update the top repo once chunk_1 is loaded (has #1 repo).
  // For now, show a placeholder from metadata.
  const topEl = document.getElementById('stat-top-star');
  topEl.textContent = '...';
  topEl.classList.add('is-text');
}

function updateStatsFromData() {
  if (state.allRepos.length === 0) return;
  document.getElementById('stat-repos').textContent = state.allRepos.length;
  const top = state.allRepos[0]; // sorted by score
  const topEl = document.getElementById('stat-top-star');
  if (top) {
    topEl.textContent = top.name || top.full_name;
    topEl.title = `${(top.stargazers_count || 0).toLocaleString()} ★`;
    topEl.classList.add('is-text');
  }
}

// ═══════════════════════════════════════════════════════════
// Table Rendering
// ═══════════════════════════════════════════════════════════
function renderTable(repos, append) {
  const tbody = document.getElementById('table-body');
  const title = document.getElementById('table-title');

  const groupLabel = state.currentGroup === 'all'
    ? 'All Technologies'
    : state.groups.find((g) => g.name === state.currentGroup)?.label || 'Results';

  if (!append) {
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
  }

  const fragment = document.createDocumentFragment();
  const startIdx = append ? tbody.querySelectorAll('.table-row').length : 0;

  repos.forEach((r, i) => {
    const tr = document.createElement('tr');
    tr.className = 'table-row row-entering';
    tr.dataset.fullname = r.full_name || '';
    tr.style.animationDelay = `${(startIdx + i) * 15}ms`;

    const score = r.score != null ? r.score.toFixed(4) : '—';
    const lang = r.language || '';
    const name = r.name || r.full_name || '';
    const url = r.html_url || '';
    const rank = startIdx + i + 1;

    tr.innerHTML = `
      <td class="cell-rank">${rank <= 3 ? `<span class="rank-badge rank-${rank}">${rank}</span>` : `<span class="rank-normal">${rank}</span>`}</td>
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

  if (append) {
    tbody.appendChild(fragment);
  } else {
    tbody.innerHTML = '';
    tbody.appendChild(fragment);
  }
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
  const isAppend = state.currentGroup === 'all' && state.loadedChunks > 1 && !query && state.currentSort.column === 'score' && state.currentSort.direction === 'desc';
  renderTable(repos, false); // full re-render for simplicity — chunks are small enough
  updateSortHeaders();
  updateStatsFromData();
  writeURLState();
  updateChartBadge();
}

// ═══════════════════════════════════════════════════════════
// Chart from top5_history.json (lightweight)
// ═══════════════════════════════════════════════════════════
async function loadChartFromTop5History() {
  const chartNote = document.getElementById('chart-note');

  try {
    const resp = await fetch(`${BASE}/data/top5_history.json`);
    if (!resp.ok) {
      chartNote.innerHTML = 'No chart data yet. Run <code>make run</code> daily to build history.';
      return;
    }
    state.top5History = await resp.json();
  } catch {
    chartNote.innerHTML = 'Chart data unavailable.';
    return;
  }

  const dates = filterDatesByRange(Object.keys(state.top5History), state.historyRange);
  if (dates.length === 0) {
    chartNote.innerHTML = 'No history data available yet. Run <code>make run</code> daily to build trend data.';
    return;
  }

  chartNote.innerHTML = dates.length < 2
    ? `Only <strong>1 day</strong> of history — showing current rankings. Trends will appear after 2+ days.`
    : `Showing <strong>${dates.length}</strong> of <strong>${Object.keys(state.top5History).length}</strong> available days.`;

  renderTrendChart(dates);
}

function filterDatesByRange(allDates, rangeDays) {
  if (!allDates || allDates.length === 0) return [];
  const sorted = [...allDates].sort().reverse();
  if (rangeDays === 0) return sorted;
  const cutoff = new Date();
  cutoff.setDate(cutoff.getDate() - rangeDays);
  const cutoffStr = cutoff.toISOString().split('T')[0];
  return sorted.filter((d) => d >= cutoffStr);
}

function updateChartBadge() {
  const badge = document.getElementById('chart-badge');
  if (!badge) return;
  badge.textContent = state.currentGroup === 'all' ? 'Top 5 Overall' : 'Top 5 in Category';
}

function getTop5RepoNames() {
  // From currently loaded data (first 5 repos = top 5 by score).
  const repos = getCurrentRepos();
  const sorted = sortRepos(repos);
  return sorted.slice(0, 5).map((r) => r.full_name);
}

function renderTrendChart(dates) {
  const ctx = document.getElementById('trend-chart').getContext('2d');
  const top5Names = getTop5RepoNames();

  const colors = [
    getComputedStyle(document.documentElement).getPropertyValue('--chart-1').trim() || '#4f46e5',
    getComputedStyle(document.documentElement).getPropertyValue('--chart-2').trim() || '#7c3aed',
    getComputedStyle(document.documentElement).getPropertyValue('--chart-3').trim() || '#059669',
    getComputedStyle(document.documentElement).getPropertyValue('--chart-4').trim() || '#d97706',
    getComputedStyle(document.documentElement).getPropertyValue('--chart-5').trim() || '#e11d48',
  ];

  const sortedDates = [...dates].sort();
  const isSingleDay = sortedDates.length === 1;

  // For single day: use bar chart. For multi-day: use line chart.
  const chartType = isSingleDay ? 'bar' : 'line';

  const datasets = top5Names.map((fullName, i) => {
    const scores = sortedDates.map((date) => {
      const dayEntries = state.top5History[date];
      if (!dayEntries) return null;
      const found = dayEntries.find((e) => e.full_name === fullName);
      return found ? found.score : null;
    });

    // Carry-forward gaps (only relevant for line charts).
    if (!isSingleDay) {
      let lastKnown = null;
      for (let j = 0; j < scores.length; j++) {
        if (scores[j] != null) lastKnown = scores[j];
        else scores[j] = lastKnown;
      }
    }

    const shortName = fullName.split('/').pop();
    return {
      label: shortName,
      data: scores,
      borderColor: colors[i % colors.length],
      backgroundColor: isSingleDay ? colors[i % colors.length] + 'cc' : colors[i % colors.length] + '20',
      borderWidth: 2,
      tension: 0.35,
      fill: false,
      pointRadius: isSingleDay ? 0 : 3,
      pointHoverRadius: 7,
      pointBackgroundColor: colors[i % colors.length],
      barPercentage: 0.8,
      categoryPercentage: 0.9,
    };
  }).filter((ds) => ds.data.some((v) => v != null));

  if (state.chart) state.chart.destroy();

  const isDark = state.theme === 'dark';
  const gridColor = isDark ? 'rgba(148, 163, 184, 0.1)' : 'rgba(0, 0, 0, 0.06)';
  const textColor = isDark ? '#94a3b8' : '#64748b';

  const xLabel = isSingleDay
    ? new Date(sortedDates[0] + 'T00:00:00').toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })
    : '';

  state.chart = new Chart(ctx, {
    type: chartType,
    data: {
      labels: isSingleDay
        ? [''] // single empty label for bar grouping
        : sortedDates.map((d) => {
            const date = new Date(d + 'T00:00:00');
            return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
          }),
      datasets,
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: {
          position: 'bottom',
          labels: { usePointStyle: true, padding: 24, color: textColor, font: { size: 13, family: "'Inter', sans-serif" } },
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
            title: () => xLabel,
            label: (ctx) => `  ${ctx.dataset.label}: ${ctx.parsed.y.toFixed(4)}`,
          },
        },
      },
      scales: {
        y: {
          min: 0, max: 1,
          ticks: { callback: (v) => v.toFixed(2), color: textColor, font: { size: 11 } },
          grid: { color: gridColor },
          title: { display: true, text: 'Popularity Score', color: textColor, font: { size: 12, weight: '600' } },
        },
        x: {
          ticks: { color: textColor, font: { size: 11 } },
          grid: { color: gridColor },
          title: { display: !isSingleDay, text: 'Date', color: textColor, font: { size: 12, weight: '600' } },
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

function refreshChart() {
  if (state.top5History) {
    const dates = filterDatesByRange(Object.keys(state.top5History), state.historyRange);
    if (dates.length >= 2) renderTrendChart(dates);
  }
}

// ═══════════════════════════════════════════════════════════
// Event Listeners
// ═══════════════════════════════════════════════════════════
function setupEventListeners() {
  // Theme toggle
  document.getElementById('theme-toggle')?.addEventListener('click', toggleTheme);

  // Date range pills
  document.getElementById('range-pills')?.addEventListener('click', (e) => {
    const pill = e.target.closest('.range-pill');
    if (!pill) return;
    document.querySelectorAll('.range-pill').forEach((p) => {
      p.classList.remove('active');
      p.setAttribute('aria-pressed', 'false');
    });
    pill.classList.add('active');
    pill.setAttribute('aria-pressed', 'true');
    state.historyRange = parseInt(pill.dataset.days, 10);
    if (state.top5History) {
      const dates = filterDatesByRange(Object.keys(state.top5History), state.historyRange);
      const note = document.getElementById('chart-note');
      if (dates.length >= 2) {
        note.innerHTML = `Showing <strong>${dates.length}</strong> of <strong>${Object.keys(state.top5History).length}</strong> available days.`;
        renderTrendChart(dates);
      }
    }
  });

  // Category tabs
  document.getElementById('category-tabs')?.addEventListener('click', async (e) => {
    const tab = e.target.closest('.tab');
    if (!tab) return;

    document.querySelectorAll('.tab').forEach((t) => {
      t.classList.remove('active');
      t.setAttribute('aria-selected', 'false');
    });
    tab.classList.add('active');
    tab.setAttribute('aria-selected', 'true');

    const group = tab.dataset.group;
    await switchToGroup(group);
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
    backBtn.classList.toggle('visible', window.scrollY > window.innerHeight * 0.75);
  });
  backBtn?.addEventListener('click', () => window.scrollTo({ top: 0, behavior: 'smooth' }));

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
    return d.toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric', hour: '2-digit', minute: '2-digit' });
  } catch { return isoString; }
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
  } catch { return isoString; }
}
