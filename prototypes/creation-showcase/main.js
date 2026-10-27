/* ══════════ 交互逻辑（状态 / 渲染 / 画布平移缩放） ══════════ */

/* ---------- Lucide 内联图标（离线硬要求） ---------- */
var ICONS = {
  sparkles: '<path d="M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z"/>',
  search: '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>',
  x: '<path d="M18 6 6 18"/><path d="m6 6 12 12"/>',
  heart: '<path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z"/>',
  play: '<polygon points="6 3 20 12 6 21 6 3"/>',
  copy: '<rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>',
  map: '<path d="M14.106 5.553a2 2 0 0 0 1.788 0l3.659-1.83A1 1 0 0 1 21 4.619v12.764a1 1 0 0 1-.553.894l-4.553 2.277a2 2 0 0 1-1.788 0l-4.212-2.106a2 2 0 0 0-1.788 0l-3.659 1.83A1 1 0 0 1 3 19.381V6.618a1 1 0 0 1 .553-.894l4.553-2.277a2 2 0 0 1 1.788 0z"/><path d="M15 5.764v15"/><path d="M9 3.236v15"/>',
  spline: '<circle cx="19" cy="5" r="2"/><circle cx="5" cy="19" r="2"/><path d="M5 17A12 12 0 0 1 17 5"/>',
  "zoom-in": '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/><line x1="11" x2="11" y1="8" y2="14"/><line x1="8" x2="14" y1="11" y2="11"/>',
  "zoom-out": '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/><line x1="8" x2="14" y1="11" y2="11"/>',
  film: '<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M7 3v18"/><path d="M3 7.5h4"/><path d="M3 12h18"/><path d="M3 16.5h4"/><path d="M17 3v18"/><path d="M17 7.5h4"/><path d="M17 16.5h4"/>',
  image: '<rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/>',
  type: '<polyline points="4 7 4 4 20 4 20 7"/><line x1="9" x2="15" y1="20" y2="20"/><line x1="12" x2="12" y1="4" y2="20"/>',
  sliders: '<line x1="21" x2="14" y1="4" y2="4"/><line x1="10" x2="3" y1="4" y2="4"/><line x1="21" x2="12" y1="12" y2="12"/><line x1="8" x2="3" y1="12" y2="12"/><line x1="21" x2="16" y1="20" y2="20"/><line x1="12" x2="3" y1="20" y2="20"/><line x1="14" x2="14" y1="2" y2="6"/><line x1="8" x2="8" y1="10" y2="14"/><line x1="16" x2="16" y1="18" y2="22"/>',
  "chevron-left": '<path d="m15 18-6-6 6-6"/>',
  eye: '<path d="M2.062 12.348a1 1 0 0 1 0-.696 10.75 10.75 0 0 1 19.876 0 1 1 0 0 1 0 .696 10.75 10.75 0 0 1-19.876 0"/><circle cx="12" cy="12" r="3"/>',
  check: '<path d="M20 6 9 17l-5-5"/>',
  clock: '<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>',
  clapperboard: '<path d="M20.2 6 3 11l-.9-2.4c-.3-1.1.3-2.2 1.3-2.5l13.5-4c1.1-.3 2.2.3 2.5 1.3Z"/><path d="m6.2 5.3 3.1 3.9"/><path d="m12.4 3.4 3.1 4"/><path d="M3 11h18v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/>'
};
var NODE_ICONS = { video: "film", image: "image", text: "type", config: "sliders" };

function svg(name, inner) {
  return '<svg viewBox="0 0 24 24" aria-hidden="true">' + (inner || ICONS[name] || "") + "</svg>";
}
function icon(name, inner) {
  return svg(name, inner);
}

/* 页面加载时把 data-icon 占位符替换为内联 SVG */
document.querySelectorAll("[data-icon]").forEach(function (el) {
  el.innerHTML = icon(el.getAttribute("data-icon"));
});

/* ---------- 全局状态 ---------- */
var state = {
  category: "全部",
  keyword: "",
  currentId: null,   // 当前详情/画布的作品
  view: "list",      // list | detail | process
  canvas: null,      // 当前画布数据
  transform: { x: 0, y: 0, k: 1 }
};

var $ = function (id) { return document.getElementById(id); };
var body = document.body;

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, function (c) {
    return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
  });
}

function avatarHtml(name, hue, cls) {
  return '<span class="avatar ' + (cls || "") + '" style="background:hsl(' + hue + ',45%,42%)">' + escapeHtml(name.slice(0, 1)) + "</span>";
}

function likeSvg(filled) {
  return svg("heart", filled ? '<path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z"/>' : null);
}

/* ---------- Toast ---------- */
var toastTimer = null;
function toast(msg) {
  var el = $("toast");
  el.innerHTML = icon("check") + escapeHtml(msg);
  el.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(function () { el.hidden = true; }, 1800);
}

/* ---------- 视图切换 ---------- */
function setView(view) {
  state.view = view;
  body.dataset.view = view;
  body.classList.toggle("no-scroll", view !== "list");
  $("detail-overlay").hidden = view !== "detail";
  $("process-overlay").hidden = view !== "process";
  if (view === "list") {
    $("detail-video").pause();
  }
  if (view === "process") {
    $("detail-video").pause();
  }
  if (view !== "process") {
    document.querySelectorAll(".canvas-node video").forEach(function (v) { v.pause(); });
  }
}

/* ---------- 列表渲染 ---------- */
function renderTabs() {
  var html = DB.categories.map(function (c) {
    return '<button class="tab' + (c === state.category ? " active" : "") + '" data-cat="' + c + '">' + c + "</button>";
  }).join("");
  $("tabs").innerHTML = html;
}

function renderGrid(works) {
  var grid = $("work-grid");
  grid.innerHTML = works.map(function (w, i) {
    return (
      '<article class="work-card" style="animation-delay:' + i * 30 + 'ms" data-id="' + w.id + '">' +
        '<div class="card-cover">' +
          '<img src="' + w.cover + '" alt="' + escapeHtml(w.title) + '" loading="lazy">' +
          '<button class="card-process-btn" data-process="' + w.id + '">' + icon("spline") + "查看创作过程</button>" +
          '<button class="card-likes' + (w.liked ? " liked" : "") + '" data-like="' + w.id + '">' + likeSvg(w.liked) + '<span class="mono">' + w.likes + "</span></button>" +
        "</div>" +
        '<h3 class="card-title">' + escapeHtml(w.title) + "</h3>" +
        '<div class="card-meta">' + avatarHtml(w.author, w.avatarHue) + "<span>" + escapeHtml(w.author) + '</span><span class="dot">·</span><span>' + escapeHtml(w.category) + "</span></div>" +
      "</article>"
    );
  }).join("");
  var empty = $("empty-state");
  empty.hidden = works.length > 0;
  grid.classList.remove("loading");
}

async function loadList() {
  $("work-grid").classList.add("loading");
  var res = await fetchShowcase({ category: state.category, keyword: state.keyword });
  renderGrid(res.data || []);
}

/* ---------- 详情弹层 ---------- */
function renderDetail(w) {
  $("detail-title").textContent = w.title;
  $("detail-video").poster = w.cover;
  $("detail-video").src = w.video;
  var av = $("detail-avatar");
  av.style.background = "hsl(" + w.avatarHue + ",45%,42%)";
  av.textContent = w.author.slice(0, 1);
  $("detail-author").textContent = w.author;
  $("detail-date").textContent = "发布于 " + w.publishedAt + " · " + w.category;
  $("detail-like-count").textContent = w.likes;
  $("detail-views").textContent = (w.views / 1000).toFixed(1) + "k";
  $("detail-duration").textContent = w.duration;
  $("detail-desc").textContent = w.desc;
  $("detail-tags").innerHTML = w.promptTags.map(function (t) { return '<span class="tag">' + escapeHtml(t) + "</span>"; }).join("");
  var likeBtn = $("detail-like");
  likeBtn.classList.toggle("liked", w.liked);
  likeBtn.querySelector(".stat-icon").innerHTML = likeSvg(w.liked);
  var followBtn = $("detail-follow");
  followBtn.classList.toggle("followed", w.followed);
  followBtn.textContent = w.followed ? "已关注" : "关注";
}

async function openDetail(id) {
  var res = await fetchWorkDetail(id);
  if (res.code !== 0) { toast(res.msg); return; }
  state.currentId = id;
  renderDetail(res.data);
  setView("detail");
}

/* ---------- 创作过程画布 ---------- */
function fmtGroupDate(label) {
  return label;
}

function renderCanvas(canvas) {
  state.canvas = canvas;
  var world = $("canvas-world");
  var svgLayer = $("edge-layer");

  // 连线
  var nodeMap = {};
  canvas.nodes.forEach(function (n) { nodeMap[n.id] = n; });
  var minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
  canvas.nodes.forEach(function (n) {
    minX = Math.min(minX, n.x); minY = Math.min(minY, n.y);
    maxX = Math.max(maxX, n.x + n.w); maxY = Math.max(maxY, n.y + n.h + 60);
  });

  var paths = canvas.edges.map(function (e) {
    var a = nodeMap[e.from], b = nodeMap[e.to];
    if (!a || !b) return "";
    var x1 = a.x + a.w, y1 = a.y + a.h / 2, x2 = b.x, y2 = b.y + b.h / 2;
    var dx = Math.max(48, (x2 - x1) * 0.5);
    return '<path class="edge-path" d="M' + x1 + " " + y1 + " C " + (x1 + dx) + " " + y1 + ", " + (x2 - dx) + " " + y2 + ", " + x2 + " " + y2 + '"/>';
  }).join("");
  svgLayer.setAttribute("width", maxX - minX + 200);
  svgLayer.setAttribute("height", maxY - minY + 200);
  svgLayer.style.left = (minX - 100) + "px";
  svgLayer.style.top = (minY - 100) + "px";
  svgLayer.setAttribute("viewBox", "0 0 " + (maxX - minX + 200) + " " + (maxY - minY + 200));
  svgLayer.innerHTML = '<g transform="translate(' + (100 - minX) + "," + (100 - minY) + ')">' + paths + "</g>";

  // 分组标题（按组内最小 y 排序）
  var groups = {};
  canvas.nodes.forEach(function (n) {
    if (!groups[n.group]) groups[n.group] = { label: n.group, count: 0, x: Infinity, y: Infinity };
    var g = groups[n.group];
    g.count++;
    g.x = Math.min(g.x, n.x);
    g.y = Math.min(g.y, n.y);
  });
  var groupHtml = Object.keys(groups).map(function (k) {
    var g = groups[k];
    var maxRes = canvas.nodes.filter(function (n) { return n.group === k && n.resolution; })[0];
    return '<div class="group-header" style="left:' + g.x + "px;top:" + (g.y - 36) + 'px">' +
      icon("play") + "<span>" + fmtGroupDate(g.label) + "(" + g.count + ")</span>" +
      (maxRes ? '<span class="gh-resolution mono">' + maxRes.resolution + "</span>" : "") +
      "</div>";
  }).join("");

  // 节点
  var nodeHtml = canvas.nodes.map(function (n) {
    var inner = "";
    if (n.type === "video") {
      inner = '<div class="node-body"><video controls preload="metadata" playsinline poster="' + (n.poster || "") + '" src="' + n.src + '"></video></div>';
    } else if (n.type === "image") {
      inner = '<div class="node-body"><img src="' + n.src + '" alt="' + escapeHtml(n.title) + '" loading="lazy"></div>';
    } else {
      inner = '<div class="node-body node-text">' + escapeHtml(n.text || "") + "</div>";
    }
    var bodyH = n.h - 28;
    return '<div class="canvas-node type-' + n.type + '" data-node="' + n.id + '" style="left:' + n.x + "px;top:" + n.y + "px;width:" + n.w + "px;height:" + n.h + 'px">' +
      '<div class="node-header">' + icon(NODE_ICONS[n.type] || "image") + "<span>" + escapeHtml(n.title) + "</span></div>" +
      '<div class="node-body-wrap" style="height:' + bodyH + 'px">' + inner + "</div>" +
      "</div>";
  }).join("");

  world.innerHTML = svgLayer.outerHTML + groupHtml + nodeHtml;
  applyTransform(state.transform.x, state.transform.y, state.transform.k, true);
  fitView();
}

function worldBounds() {
  var ns = (state.canvas || {}).nodes || [];
  if (!ns.length) return { x: 0, y: 0, w: 1000, h: 600 };
  var minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity;
  ns.forEach(function (n) {
    minX = Math.min(minX, n.x); minY = Math.min(minY, n.y);
    maxX = Math.max(maxX, n.x + n.w); maxY = Math.max(maxY, n.y + n.h + 60);
  });
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY };
}

function fitView() {
  var vp = $("canvas-viewport");
  var b = worldBounds();
  var pad = 80;
  var k = Math.min((vp.clientWidth - pad * 2) / b.w, (vp.clientHeight - pad * 2) / b.h, 1);
  k = Math.max(0.4, Math.min(2, k));
  var x = (vp.clientWidth - b.w * k) / 2 - b.x * k;
  var y = (vp.clientHeight - b.h * k) / 2 - b.y * k;
  applyTransform(x, y, k, false);
}

function applyTransform(x, y, k, skipZoomLabel) {
  var t = state.transform;
  t.x = x; t.y = y; t.k = k;
  var world = $("canvas-world");
  world.style.transform = "translate(" + x + "px," + y + "px) scale(" + k + ")";
  var vp = $("canvas-viewport");
  vp.style.backgroundSize = 24 * k + "px " + 24 * k + "px";
  vp.style.backgroundPosition = x + "px " + y + "px";
  if (!skipZoomLabel) $("zoom-value").textContent = Math.round(k * 100) + "%";
  drawMinimap();
}

function zoomAt(cx, cy, factor) {
  var t = state.transform;
  var k = Math.max(0.4, Math.min(2, t.k * factor));
  applyTransform(cx - (cx - t.x) * (k / t.k), cy - (cy - t.y) * (k / t.k), k);
}

async function openProcess(id) {
  var res = await fetchWorkProcess(id);
  if (res.code !== 0) { toast(res.msg); return; }
  var workRes = await fetchWorkDetail(id);
  if (workRes.code === 0) {
    state.currentId = id;
    renderDetail(workRes.data); // 供「返回预览」使用
  }
  $("process-title").textContent = (workRes.data || {}).title || "创作过程";
  setView("process");
  renderCanvas(res.data);
}

/* ---------- 小地图 ---------- */
var MINIMAP_W = 180, MINIMAP_H = 112;
function drawMinimap() {
  var panel = $("minimap-panel");
  if (panel.hidden) return;
  var canvas = $("minimap");
  var ctx = canvas.getContext("2d");
  ctx.clearRect(0, 0, MINIMAP_W, MINIMAP_H);
  ctx.fillStyle = "#1f1d1a";
  ctx.fillRect(0, 0, MINIMAP_W, MINIMAP_H);
  var b = worldBounds();
  var scale = Math.min((MINIMAP_W - 16) / b.w, (MINIMAP_H - 16) / b.h);
  var ox = (MINIMAP_W - b.w * scale) / 2 - b.x * scale;
  var oy = (MINIMAP_H - b.h * scale) / 2 - b.y * scale;
  (state.canvas || { nodes: [] }).nodes.forEach(function (n) {
    ctx.fillStyle = n.type === "video" ? "rgba(245,245,244,.5)" : "rgba(245,245,244,.25)";
    ctx.fillRect(ox + n.x * scale, oy + n.y * scale, Math.max(3, n.w * scale), Math.max(2, n.h * scale));
  });
  var vp = $("canvas-viewport");
  var t = state.transform;
  ctx.strokeStyle = "#fafaf9";
  ctx.lineWidth = 1.2;
  ctx.strokeRect(ox - t.x / t.k * scale, oy - t.y / t.k * scale, vp.clientWidth / t.k * scale, vp.clientHeight / t.k * scale);
}

/* ---------- 事件绑定 ---------- */
function bindEvents() {
  // 顶栏
  $("btn-my-canvas").addEventListener("click", function () { toast("演示原型：跳转我的画布"); });
  $("btn-publish").addEventListener("click", function () { toast("演示原型：从画布项目一键发布"); });

  // 分类 tab
  $("tabs").addEventListener("click", function (e) {
    var btn = e.target.closest("[data-cat]");
    if (!btn) return;
    state.category = btn.dataset.cat;
    renderTabs();
    loadList();
  });

  // 搜索（防抖）
  var searchTimer = null;
  $("search-input").addEventListener("input", function (e) {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(function () {
      state.keyword = e.target.value;
      loadList();
    }, 250);
  });
  $("btn-clear-search").addEventListener("click", function () {
    $("search-input").value = "";
    state.keyword = "";
    loadList();
  });

  // 卡片：点击开详情 / 查看过程 / 点赞
  $("work-grid").addEventListener("click", function (e) {
    var likeBtn = e.target.closest("[data-like]");
    if (likeBtn) {
      e.stopPropagation();
      handleLike(likeBtn.dataset.like);
      return;
    }
    var processBtn = e.target.closest("[data-process]");
    if (processBtn) {
      e.stopPropagation();
      openProcess(processBtn.dataset.process);
      return;
    }
    var card = e.target.closest(".work-card");
    if (card) openDetail(card.dataset.id);
  });

  // 详情
  $("detail-back").addEventListener("click", function () { setView("list"); });
  $("detail-close").addEventListener("click", function () { setView("list"); });
  $("detail-fork").addEventListener("click", function () { handleFork(state.currentId); });
  $("detail-like").addEventListener("click", function () { handleLike(state.currentId); });
  $("detail-follow").addEventListener("click", function () { handleFollow(state.currentId); });
  $("detail-open-process").addEventListener("click", function () { openProcess(state.currentId); });

  // 创作过程
  $("process-back").addEventListener("click", function () { setView("detail"); });
  $("process-close").addEventListener("click", function () { setView("list"); });
  $("process-fork").addEventListener("click", function () { handleFork(state.currentId); });
  $("tool-minimap").addEventListener("click", function () {
    var panel = $("minimap-panel");
    panel.hidden = !panel.hidden;
    $("tool-minimap").classList.toggle("active", !panel.hidden);
    if (!panel.hidden) drawMinimap();
  });
  $("tool-edges").addEventListener("click", function () {
    var paths = document.querySelectorAll("#edge-layer .edge-path");
    var hide = paths.length && !paths[0].classList.contains("hidden");
    paths.forEach(function (p) { p.classList.toggle("hidden", hide); });
    $("tool-edges").classList.toggle("active", hide);
  });
  $("tool-zoom-in").addEventListener("click", function () {
    var vp = $("canvas-viewport");
    zoomAt(vp.clientWidth / 2, vp.clientHeight / 2, 1.2);
  });
  $("tool-zoom-out").addEventListener("click", function () {
    var vp = $("canvas-viewport");
    zoomAt(vp.clientWidth / 2, vp.clientHeight / 2, 1 / 1.2);
  });
  $("minimap-panel").addEventListener("click", function (e) {
    var rect = $("minimap").getBoundingClientRect();
    var b = worldBounds();
    var scale = Math.min((MINIMAP_W - 16) / b.w, (MINIMAP_H - 16) / b.h);
    var ox = (MINIMAP_W - b.w * scale) / 2 - b.x * scale;
    var oy = (MINIMAP_H - b.h * scale) / 2 - b.y * scale;
    var wx = (e.clientX - rect.left - ox) / scale;
    var wy = (e.clientY - rect.top - oy) / scale;
    var vp = $("canvas-viewport");
    var t = state.transform;
    applyTransform(vp.clientWidth / 2 - wx * t.k, vp.clientHeight / 2 - wy * t.k, t.k);
  });

  // 画布：滚轮缩放 + 拖拽平移
  var vp = $("canvas-viewport");
  vp.addEventListener("wheel", function (e) {
    e.preventDefault();
    var rect = vp.getBoundingClientRect();
    zoomAt(e.clientX - rect.left, e.clientY - rect.top, e.deltaY < 0 ? 1.1 : 1 / 1.1);
  }, { passive: false });

  var panning = null;
  vp.addEventListener("pointerdown", function (e) {
    if (e.button !== 0) return;
    if (e.target.closest("video, button, .minimap-panel")) return; // 视频控件等不触发平移
    panning = { sx: e.clientX, sy: e.clientY, tx: state.transform.x, ty: state.transform.y };
    vp.classList.add("panning");
    vp.setPointerCapture(e.pointerId);
  });
  vp.addEventListener("pointermove", function (e) {
    if (!panning) return;
    applyTransform(panning.tx + e.clientX - panning.sx, panning.ty + e.clientY - panning.sy, state.transform.k);
  });
  var endPan = function () { panning = null; vp.classList.remove("panning"); };
  vp.addEventListener("pointerup", endPan);
  vp.addEventListener("pointercancel", endPan);

  // Esc 关闭当前弹层
  document.addEventListener("keydown", function (e) {
    if (e.key !== "Escape") return;
    if (state.view === "process") setView("detail");
    else if (state.view === "detail") setView("list");
  });

  // 窗口尺寸变化：重置适配
  window.addEventListener("resize", function () {
    if (state.view === "process") fitView();
  });
}

/* ---------- 点赞 / 关注 / 复制 ---------- */
async function handleLike(id) {
  if (!id) return;
  var res = await toggleLike(id);
  if (res.code !== 0) return;
  var work = DB.works.find(function (w) { return w.id === id; });
  // 同步列表卡片
  var cardLike = document.querySelector('[data-like="' + id + '"]');
  if (cardLike) {
    cardLike.classList.toggle("liked", work.liked);
    cardLike.querySelector("span.mono").textContent = work.likes;
  }
  if (state.currentId === id) {
    var likeBtn = $("detail-like");
    likeBtn.classList.toggle("liked", work.liked);
    likeBtn.querySelector(".stat-icon").innerHTML = likeSvg(work.liked);
    $("detail-like-count").textContent = work.likes;
  }
}

async function handleFollow(id) {
  var res = await toggleFollow(id);
  if (res.code !== 0) return;
  var work = DB.works.find(function (w) { return w.id === id; });
  var btn = $("detail-follow");
  btn.classList.toggle("followed", work.followed);
  btn.textContent = work.followed ? "已关注" : "关注";
}

async function handleFork(id) {
  if (!id) return;
  toast("正在复制到我的画布…");
  var res = await forkWork(id);
  if (res.code === 0) toast("已复制到我的画布（演示）");
  else toast(res.msg);
}

/* ---------- 启动 ---------- */
renderTabs();
bindEvents();
loadList();
