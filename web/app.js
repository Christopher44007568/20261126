const state = {
  scene: null,
  maps: new Map(),
  filter: "all",
  textures: new Map(),
  patterns: new Map(),
};

const $ = (selector, root = document) => root.querySelector(selector);
const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function pad(value, width = 2) {
  return String(value).padStart(width, "0");
}

function statusClass(item) {
  if (item.placeholder) return "placeholder";
  return item.status || "asset";
}

function statusLabel(item) {
  return item.placeholder ? "棋盘格占位" : item.statusLabel;
}

async function boot() {
  bindInspectorEvents();
  bindFilters();
  setupCanvas();
  try {
    const response = await fetch("/api/scene", { cache: "no-store" });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    state.scene = await response.json();
    renderSceneData();
  } catch (error) {
    renderLoadError(error);
  }
}

function renderSceneData() {
  const scene = state.scene;
  renderMetrics(scene);
  renderOrder(scene.searchOrder);
  renderStatusGroups(scene);
  renderMaterials(scene.materials);
  renderChrome(scene);
  loadSceneTextures(scene);
  drawScene();
}

function renderMetrics(scene) {
  const metrics = [
    { value: pad(scene.summary.total), label: "贴图槽位" },
    { value: pad(scene.summary.resolved), label: "正常解析" },
    { value: pad(scene.summary.placeholder), label: "棋盘占位" },
  ];
  $("#hero-metrics").innerHTML = metrics.map((item) => `
    <div class="metric">
      <strong>${escapeHTML(item.value)}</strong>
      <span>${escapeHTML(item.label)}</span>
    </div>
  `).join("");
}

function renderOrder(steps) {
  const accent = {
    asset: "var(--amber)",
    cache: "var(--mint)",
    fallback: "var(--orange)",
    config: "var(--orange)",
  };
  const entries = [];
  steps.forEach((step, index) => {
    if (index > 0) entries.push('<div class="order-arrow" aria-hidden="true">→</div>');
    const duplicates = step.duplicates > 0
      ? `<span class="dup">${step.duplicates} 个同名候选被跳过</span>`
      : `<span>无后续同名文件</span>`;
    entries.push(`
      <article class="order-step" style="--accent:${accent[step.kind] || accent.asset}">
        <span class="step-number">${pad(index + 1)}</span>
        <div class="step-body">
          <strong>${escapeHTML(step.label)}</strong>
          <code>${escapeHTML(step.path)}</code>
        </div>
        <span class="step-state">${escapeHTML(step.state)}</span>
        <div class="step-stats">
          <b>${step.hits}</b> 最终采用
          ${duplicates}
        </div>
      </article>
    `);
  });
  $("#order-rail").innerHTML = entries.join("");
}

function renderStatusGroups(scene) {
  const states = new Map(scene.searchOrder.map((step) => [step.id, step.state]));
  const total = Math.max(1, scene.summary.total);
  $("#status-stack").innerHTML = scene.groups.map((group, index) => {
    const extra = group.kind === "fallback" || group.kind === "config"
      ? `<span><b>${group.fileCount}</b> 文件命中</span><span class="warn"><b>${group.placeholderCount}</b> 棋盘占位</span>`
      : `<span><b>${group.count}</b> 槽位接管</span>${group.conflictCount ? `<span class="warn"><b>${group.conflictCount}</b> 同名冲突</span>` : "<span>无同名冲突</span>"}`;
    return `
      <article class="status-card" style="--accent:${group.accent};--fill:${Math.max(8, group.count / total * 100)}%;animation-delay:${index * 70}ms">
        <div class="status-top">
          <span class="status-index">/ ${pad(index + 1)}</span>
          <span class="status-state">${escapeHTML(states.get(group.id) || group.kind)}</span>
        </div>
        <div class="status-main">
          <h3>${escapeHTML(group.label)}</h3>
          <strong>${pad(group.count)}<small> / ${pad(scene.summary.total)}</small></strong>
        </div>
        <div class="status-path" title="${escapeHTML(group.path)}">${escapeHTML(group.path)}</div>
        <p class="status-desc">${escapeHTML(group.description)}</p>
        <div class="status-meter"><span></span></div>
        <div class="status-meta">${extra}</div>
      </article>
    `;
  }).join("");
}

function renderMaterials(materials) {
  state.maps.clear();
  $("#materials").innerHTML = materials.map((material, index) => {
    const tiles = material.maps.map((item) => {
      const key = `${material.id}:${item.filename}`;
      state.maps.set(key, { material, item });
      const conflicts = item.conflicts?.length || 0;
      return `
        <button class="map-tile" type="button" data-map-key="${escapeHTML(key)}"
          data-conflict="${conflicts > 0}" data-placeholder="${Boolean(item.placeholder)}">
          <span class="map-preview">
            <img src="${escapeHTML(item.url)}" alt="${escapeHTML(material.name + " " + item.mapLabel)}" loading="eager">
            <span class="map-badge ${statusClass(item)}">${escapeHTML(statusLabel(item))}</span>
            ${item.placeholder ? '<span class="placeholder-mark">CHECKER</span>' : ""}
            <span class="map-type">${escapeHTML(item.mapLabel)}</span>
          </span>
          <span class="map-info">
            <span class="map-filename">${escapeHTML(item.filename)}</span>
            <span class="map-source" title="${escapeHTML(item.source)}">${escapeHTML(item.source)}</span>
            ${conflicts ? `<span class="map-conflict">${conflicts} 个同名候选 · 已抑制</span>` : ""}
          </span>
        </button>
      `;
    }).join("");

    const tally = `
      <div class="material-tally">
        <span class="tally"><strong>${material.resolved}</strong> / 3 已解析</span>
        ${material.placeholder ? `<span class="tally warn"><strong>${material.placeholder}</strong> 棋盘占位</span>` : ""}
        ${material.maps.some((item) => item.conflicts?.length) ? '<span class="tally warn">存在同名覆盖</span>' : ""}
      </div>
    `;
    return `
      <article class="material-card" data-material-id="${escapeHTML(material.id)}"
        style="--accent:${material.accent};animation-delay:${index * 65}ms">
        <div class="material-intro">
          <div class="material-number">${escapeHTML(material.eyebrow)}</div>
          <h3>${escapeHTML(material.name)}</h3>
          <p>${escapeHTML(material.description)}</p>
          ${tally}
        </div>
        <div class="map-grid">${tiles}</div>
      </article>
    `;
  }).join("");
}

function renderChrome(scene) {
  const title = $("#scene-caption-title");
  const sub = $("#scene-caption-sub");
  if (title) {
    title.textContent = `${scene.materials.length} 组材质 / ${scene.summary.total} 个贴图槽位`;
  }
  if (sub) {
    sub.textContent = `${scene.summary.placeholder} 个棋盘占位 · 场景保持可见`;
  }
  const generated = $("#generated-at");
  if (generated) {
    generated.textContent = `RESOLVED ${scene.generatedAt.replace("T", " ").slice(0, 19)} UTC`;
  }
}

function renderLoadError(error) {
  $("#materials").innerHTML = `
    <div class="empty-state">
      无法读取解析结果：${escapeHTML(error.message)}
    </div>
  `;
  $("#status-stack").innerHTML = "";
  $("#order-rail").innerHTML = "";
  $("#scene-caption-title").textContent = "解析服务不可用";
  $("#scene-caption-sub").textContent = "请从 Go 本地程序访问";
}

function bindFilters() {
  $$(".filter-button").forEach((button) => {
    button.addEventListener("click", () => {
      state.filter = button.dataset.filter;
      $$(".filter-button").forEach((item) => item.classList.toggle("is-active", item === button));
      applyFilter();
    });
  });
}

function applyFilter() {
  $$(".material-card").forEach((card) => {
    let visible = 0;
    $$(".map-tile", card).forEach((tile) => {
      const show = state.filter === "all"
        || (state.filter === "conflict" && tile.dataset.conflict === "true")
        || (state.filter === "placeholder" && tile.dataset.placeholder === "true");
      tile.classList.toggle("is-hidden", !show);
      if (show) visible++;
    });
    card.classList.toggle("is-hidden", visible === 0);
  });
}

function bindInspectorEvents() {
  $("#materials").addEventListener("click", (event) => {
    const tile = event.target.closest(".map-tile");
    if (!tile) return;
    openInspector(tile.dataset.mapKey);
  });
  $$("[data-close-inspector]").forEach((element) => {
    element.addEventListener("click", closeInspector);
  });
  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") closeInspector();
  });
}

function openInspector(key) {
  const record = state.maps.get(key);
  if (!record) return;
  const { material, item } = record;
  const selectedPath = item.selected?.path || "";
  const trace = (item.trace || []).map((entry, index) => {
    const selected = entry.exists && entry.path === selectedPath;
    const conflict = entry.exists && !selected;
    const result = selected ? "采用" : conflict ? "抑制" : "缺失";
    return `
      <li class="trace-item ${selected ? "is-selected" : ""} ${conflict ? "is-conflict" : ""}">
        <span class="trace-priority">#${pad(index + 1)}</span>
        <span class="trace-path">
          <strong title="${escapeHTML(entry.path)}">${escapeHTML(entry.path)}</strong>
          <span>${escapeHTML(entry.rootLabel)} · ${escapeHTML(entry.kind)}</span>
        </span>
        <span class="trace-result">${result}</span>
      </li>
    `;
  }).join("");

  const conflicts = (item.conflicts || []).map((candidate) => `
    <li class="candidate-item">
      <strong>${escapeHTML(candidate.path)}</strong>
      <span>优先级 #${pad(candidate.priority)} · ${escapeHTML(candidate.rootLabel)} · 已存在但未采用</span>
    </li>
  `).join("");

  const sourceKind = item.placeholder ? "PROCEDURAL FALLBACK" : (item.selected?.rootLabel || "").toUpperCase();
  const statusText = item.placeholder
    ? "所有目录均未找到文件。程序生成棋盘格，避免黑色材质中断场景。"
    : `从优先级 #${pad(item.selected?.priority || 1)} 的 ${item.selected?.rootLabel || "路径"} 采用此文件。`;
  $("#inspector-content").innerHTML = `
    <div class="inspector-kicker">${escapeHTML(material.eyebrow)} / ${escapeHTML(item.mapLabel)}</div>
    <h2 class="inspector-title">${escapeHTML(item.filename)}</h2>
    <p class="inspector-sub">${escapeHTML(material.name)} · ${escapeHTML(statusLabel(item))}</p>
    <div class="inspector-preview">
      <img src="${escapeHTML(item.url)}" alt="${escapeHTML(item.filename)} 预览">
      <span class="map-badge ${statusClass(item)}">${escapeHTML(statusLabel(item))}</span>
    </div>
    <div class="inspector-section">
      <h3>最终采用来源</h3>
      <div class="source-box ${item.placeholder ? "is-placeholder" : ""}">
        <div class="source-box-label">${escapeHTML(sourceKind)}</div>
        <strong>${escapeHTML(item.source)}</strong>
        <span>${escapeHTML(statusText)}</span>
      </div>
      <p class="path-rule">搜索命中后立即锁定；优先级更低的同名文件只记录，不覆盖最终来源。</p>
    </div>
    <div class="inspector-section">
      <h3>路径搜索轨迹</h3>
      <ol class="trace-list">${trace}</ol>
    </div>
    ${conflicts ? `
      <div class="inspector-section">
        <h3>同名候选 · 已抑制</h3>
        <p class="candidate-note">这些文件与最终采用项文件名相同，但位于较低优先级目录。</p>
        <ul class="candidate-list">${conflicts}</ul>
      </div>
    ` : ""}
  `;
  const inspector = $("#inspector");
  inspector.classList.add("is-open");
  inspector.setAttribute("aria-hidden", "false");
  document.body.style.overflow = "hidden";
  $(".inspector-close").focus({ preventScroll: true });
}

function closeInspector() {
  const inspector = $("#inspector");
  if (!inspector.classList.contains("is-open")) return;
  inspector.classList.remove("is-open");
  inspector.setAttribute("aria-hidden", "true");
  document.body.style.overflow = "";
}

const canvasState = {
  canvas: null,
  ctx: null,
  width: 0,
  height: 0,
  ready: false,
};

function setupCanvas() {
  const canvas = $("#scene-canvas");
  if (!canvas) return;
  const ctx = canvas.getContext("2d");
  canvasState.canvas = canvas;
  canvasState.ctx = ctx;
  const resize = () => {
    const rect = canvas.getBoundingClientRect();
    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    canvasState.width = Math.max(1, rect.width);
    canvasState.height = Math.max(1, rect.height);
    canvas.width = Math.round(canvasState.width * dpr);
    canvas.height = Math.round(canvasState.height * dpr);
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    canvasState.ready = true;
    drawScene();
  };
  new ResizeObserver(resize).observe(canvas);
  resize();
}

function loadSceneTextures(scene) {
  scene.materials.forEach((material) => {
    const albedo = material.maps.find((item) => item.kind === "albedo") || material.maps[0];
    if (albedo) ensureTexture(albedo.url);
  });
}

function ensureTexture(url) {
  if (!url || state.textures.has(url)) return;
  const image = new Image();
  state.textures.set(url, image);
  image.onload = () => {
    state.patterns.delete(url);
    drawScene();
  };
  image.onerror = () => drawScene();
  image.src = url;
}

function checkerPattern(ctx) {
  if (state.patterns.has("checker")) return state.patterns.get("checker");
  const tile = document.createElement("canvas");
  tile.width = 96;
  tile.height = 96;
  const paint = tile.getContext("2d");
  for (let y = 0; y < 4; y++) {
    for (let x = 0; x < 4; x++) {
      paint.fillStyle = (x + y) % 2 ? "#625d54" : "#ded6c5";
      paint.fillRect(x * 24, y * 24, 24, 24);
    }
  }
  paint.strokeStyle = "rgba(232,139,74,.85)";
  paint.lineWidth = 4;
  paint.strokeRect(2, 2, 92, 92);
  const pattern = ctx.createPattern(tile, "repeat");
  state.patterns.set("checker", pattern);
  return pattern;
}

function texturePattern(ctx, url) {
  if (state.patterns.has(url)) return state.patterns.get(url);
  const image = state.textures.get(url);
  if (!image || !image.complete || !image.naturalWidth) return null;
  const tile = document.createElement("canvas");
  tile.width = 128;
  tile.height = 128;
  const paint = tile.getContext("2d");
  paint.drawImage(image, 0, 0, 128, 128);
  const pattern = ctx.createPattern(tile, "repeat");
  state.patterns.set(url, pattern);
  return pattern;
}

function drawScene() {
  if (!canvasState.ready) return;
  const ctx = canvasState.ctx;
  const w = canvasState.width;
  const h = canvasState.height;
  ctx.clearRect(0, 0, w, h);
  drawBackdrop(ctx, w, h);
  const materials = state.scene?.materials || [];
  const centers = [w * 0.225, w * 0.5, w * 0.775];
  const baseY = h * 0.59;
  const cubeWidth = Math.max(56, Math.min(124, w * 0.142));
  materials.forEach((material, index) => {
    const albedo = material.maps.find((item) => item.kind === "albedo") || material.maps[0];
    const item = albedo || {};
    const pattern = texturePattern(ctx, item.url) || checkerPattern(ctx);
    drawPlinth(ctx, centers[index], baseY, cubeWidth, material.accent);
    drawCube(ctx, centers[index], baseY, cubeWidth, pattern, material.accent);
    drawSceneLabel(ctx, centers[index], baseY, cubeWidth, material, item);
  });
}

function drawBackdrop(ctx, w, h) {
  const sky = ctx.createLinearGradient(0, 0, 0, h);
  sky.addColorStop(0, "#0b1411");
  sky.addColorStop(0.58, "#14231c");
  sky.addColorStop(1, "#1b2d24");
  ctx.fillStyle = sky;
  ctx.fillRect(0, 0, w, h);

  const horizon = h * 0.59;
  const glow = ctx.createRadialGradient(w * 0.5, horizon * 0.68, 10, w * 0.5, horizon * 0.68, w * 0.58);
  glow.addColorStop(0, "rgba(123, 166, 135, 0.16)");
  glow.addColorStop(0.55, "rgba(72, 113, 88, 0.04)");
  glow.addColorStop(1, "rgba(0, 0, 0, 0)");
  ctx.fillStyle = glow;
  ctx.fillRect(0, 0, w, h);

  [w * 0.225, w * 0.5, w * 0.775].forEach((x) => {
    const cone = ctx.createLinearGradient(0, 0, 0, horizon + 20);
    cone.addColorStop(0, "rgba(212, 230, 209, 0.055)");
    cone.addColorStop(1, "rgba(212, 230, 209, 0)");
    ctx.fillStyle = cone;
    ctx.beginPath();
    ctx.moveTo(x - 32, 0);
    ctx.lineTo(x + 32, 0);
    ctx.lineTo(x + 110, horizon + 10);
    ctx.lineTo(x - 110, horizon + 10);
    ctx.closePath();
    ctx.fill();
  });

  ctx.strokeStyle = "rgba(226, 237, 223, 0.08)";
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(0, horizon);
  ctx.lineTo(w, horizon);
  ctx.stroke();

  const floor = ctx.createLinearGradient(0, horizon, 0, h);
  floor.addColorStop(0, "rgba(38, 61, 49, 0.28)");
  floor.addColorStop(1, "rgba(7, 13, 10, 0.82)");
  ctx.fillStyle = floor;
  ctx.fillRect(0, horizon, w, h - horizon);

  ctx.save();
  ctx.strokeStyle = "rgba(189, 214, 193, 0.075)";
  ctx.lineWidth = 1;
  for (let i = -7; i <= 7; i++) {
    ctx.beginPath();
    ctx.moveTo(w * 0.5, horizon);
    ctx.lineTo(w * 0.5 + i * w * 0.19, h);
    ctx.stroke();
  }
  for (let i = 1; i <= 7; i++) {
    const t = i / 7;
    const y = horizon + Math.pow(t, 1.75) * (h - horizon);
    ctx.globalAlpha = 0.34 + t * 0.58;
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(w, y);
    ctx.stroke();
  }
  ctx.restore();
}

function drawPlinth(ctx, x, baseY, cubeWidth, accent) {
  const width = cubeWidth * 1.52;
  const topY = baseY + 7;
  const bottomY = baseY + 45;
  const ellipse = width * 0.19;

  const shadow = ctx.createRadialGradient(x, bottomY + 10, 3, x, bottomY + 10, width * 0.72);
  shadow.addColorStop(0, "rgba(0, 0, 0, 0.42)");
  shadow.addColorStop(1, "rgba(0, 0, 0, 0)");
  ctx.fillStyle = shadow;
  ctx.beginPath();
  ctx.ellipse(x, bottomY + 9, width * 0.72, 13, 0, 0, Math.PI * 2);
  ctx.fill();

  ctx.fillStyle = "#17271f";
  ctx.beginPath();
  ctx.moveTo(x - width / 2, topY);
  ctx.lineTo(x - width / 2, bottomY - 7);
  ctx.bezierCurveTo(x - width / 2, bottomY + 8, x + width / 2, bottomY + 8, x + width / 2, bottomY - 7);
  ctx.lineTo(x + width / 2, topY);
  ctx.closePath();
  ctx.fill();

  ctx.fillStyle = "#263b30";
  ctx.beginPath();
  ctx.ellipse(x, topY, width / 2, ellipse, 0, 0, Math.PI * 2);
  ctx.fill();

  ctx.strokeStyle = "rgba(222, 235, 219, 0.12)";
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.ellipse(x, topY, width / 2, ellipse, 0, 0, Math.PI * 2);
  ctx.stroke();

  ctx.strokeStyle = accent;
  ctx.globalAlpha = 0.34;
  ctx.beginPath();
  ctx.ellipse(x, topY, width * 0.33, ellipse * 0.64, 0, 0, Math.PI * 2);
  ctx.stroke();
  ctx.globalAlpha = 1;
}

function drawCube(ctx, x, baseY, width, pattern, accent) {
  const left = x - width / 2;
  const right = x + width / 2;
  const bottom = baseY + 2;
  const top = bottom - width * 1.02;
  const dx = width * 0.28;
  const dy = width * 0.23;
  const topFace = [
    { x: left, y: top },
    { x: right, y: top },
    { x: right + dx, y: top - dy },
    { x: left + dx, y: top - dy },
  ];
  const sideFace = [
    { x: right, y: top },
    { x: right + dx, y: top - dy },
    { x: right + dx, y: bottom - dy },
    { x: right, y: bottom },
  ];
  const frontFace = [
    { x: left, y: top },
    { x: right, y: top },
    { x: right, y: bottom },
    { x: left, y: bottom },
  ];

  drawTexturedFace(ctx, frontFace, pattern, "front", accent);
  drawTexturedFace(ctx, sideFace, pattern, "side", accent);
  drawTexturedFace(ctx, topFace, pattern, "top", accent);

  ctx.strokeStyle = "rgba(242, 239, 226, 0.19)";
  ctx.lineWidth = 1;
  [frontFace, sideFace, topFace].forEach((face) => {
    ctx.beginPath();
    face.forEach((point, index) => {
      if (index === 0) ctx.moveTo(point.x, point.y);
      else ctx.lineTo(point.x, point.y);
    });
    ctx.closePath();
    ctx.stroke();
  });

  ctx.strokeStyle = accent;
  ctx.globalAlpha = 0.9;
  ctx.lineWidth = 2;
  ctx.beginPath();
  ctx.moveTo(left, bottom);
  ctx.lineTo(right, bottom);
  ctx.stroke();
  ctx.globalAlpha = 1;
}

function drawTexturedFace(ctx, points, pattern, side, accent) {
  const xs = points.map((point) => point.x);
  const ys = points.map((point) => point.y);
  const minX = Math.min(...xs);
  const minY = Math.min(...ys);
  const maxX = Math.max(...xs);
  const maxY = Math.max(...ys);

  ctx.save();
  ctx.beginPath();
  points.forEach((point, index) => {
    if (index === 0) ctx.moveTo(point.x, point.y);
    else ctx.lineTo(point.x, point.y);
  });
  ctx.closePath();
  ctx.clip();
  ctx.translate(minX, minY);
  ctx.fillStyle = pattern;
  ctx.fillRect(0, 0, maxX - minX, maxY - minY);

  const shade = ctx.createLinearGradient(0, 0, maxX - minX, maxY - minY);
  if (side === "top") {
    shade.addColorStop(0, "rgba(255, 244, 216, 0.19)");
    shade.addColorStop(1, "rgba(255, 255, 255, 0.02)");
  } else if (side === "side") {
    shade.addColorStop(0, "rgba(3, 8, 5, 0.25)");
    shade.addColorStop(1, "rgba(3, 8, 5, 0.48)");
  } else {
    shade.addColorStop(0, "rgba(255, 255, 255, 0.10)");
    shade.addColorStop(0.55, "rgba(0, 0, 0, 0)");
    shade.addColorStop(1, "rgba(0, 0, 0, 0.27)");
  }
  ctx.fillStyle = shade;
  ctx.fillRect(0, 0, maxX - minX, maxY - minY);

  const reflection = ctx.createLinearGradient(0, (maxY - minY) * 0.45, 0, maxY - minY);
  reflection.addColorStop(0, "rgba(255, 255, 255, 0)");
  reflection.addColorStop(1, "rgba(0, 0, 0, 0.12)");
  ctx.fillStyle = reflection;
  ctx.fillRect(0, 0, maxX - minX, maxY - minY);
  ctx.restore();

  if (side === "front") {
    ctx.strokeStyle = accent;
    ctx.globalAlpha = 0.14;
    ctx.lineWidth = 3;
    ctx.stroke();
    ctx.globalAlpha = 1;
  }
}

function drawSceneLabel(ctx, x, baseY, cubeWidth, material, item) {
  const y = baseY + 76;
  const placeholder = Boolean(item.placeholder);
  const status = placeholder
    ? "CHECKER FALLBACK"
    : (item.selected?.rootLabel || item.statusLabel || "RESOLVED").toUpperCase();
  ctx.textAlign = "center";
  ctx.fillStyle = material.accent;
  ctx.fillRect(x - 9, y - 16, 18, 1);
  ctx.font = '600 11px "Avenir Next", "PingFang SC", sans-serif';
  ctx.fillStyle = "rgba(246, 242, 231, 0.88)";
  ctx.fillText(material.name, x, y);
  ctx.font = '7px "SFMono-Regular", "Cascadia Code", monospace';
  ctx.fillStyle = placeholder ? "rgba(232, 139, 74, 0.9)" : "rgba(225, 233, 222, 0.4)";
  ctx.fillText(status, x, y + 12);
  ctx.textAlign = "left";
}

boot();
