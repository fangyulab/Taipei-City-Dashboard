<script setup>
import { ref, computed, onBeforeUnmount, nextTick, watch } from "vue";
import { useDialogStore } from "../../store/dialogStore";
import DialogContainer from "./DialogContainer.vue";
import mapboxgl from "mapbox-gl";

const dialogStore = useDialogStore();

// 表單狀態
const address = ref("");
const areaValue = ref("");
const roofType = ref("");
const roofDirection = ref("");
const showResult = ref(false);
const isCalculating = ref(false);
const apiError = ref(false);

// 地圖
const mapContainer = ref(null);
let map = null;

// 自製繪圖狀態
let drawPoints = [];
let isDrawing = ref(false);
let markers = [];

// 屋頂類型選項
const roofTypes = ["平屋頂", "斜屋頂", "金屬浪板", "其他"];
const directions = ["南向", "東南向", "西南向", "東向", "西向", "北向"];

// ========== 自製面積計算（取代 @turf/area）==========
function calculatePolygonArea(coords) {
  // Shoelace formula on spherical coordinates
  const toRad = (deg) => (deg * Math.PI) / 180;
  const R = 6371000;
  let area = 0;
  const n = coords.length;
  for (let i = 0; i < n; i++) {
    const j = (i + 1) % n;
    area +=
      toRad(coords[j][0] - coords[i][0]) *
      (2 + Math.sin(toRad(coords[i][1])) + Math.sin(toRad(coords[j][1])));
  }
  return Math.abs((area * R * R) / 2);
}

// ========== 自製多邊形繪製（取代 @mapbox/mapbox-gl-draw）==========
function updateDrawLayers() {
  if (!map) return;

  // 更新點 source
  const pointFeatures = drawPoints.map((p) => ({
    type: "Feature",
    geometry: { type: "Point", coordinates: p },
  }));

  if (map.getSource("draw-points")) {
    map.getSource("draw-points").setData({
      type: "FeatureCollection",
      features: pointFeatures,
    });
  }

  // 更新線/面 source
  if (drawPoints.length >= 2) {
    const lineCoords = [...drawPoints];
    if (drawPoints.length >= 3) {
      lineCoords.push(drawPoints[0]); // 閉合
    }

    const feature =
      drawPoints.length >= 3
        ? {
            type: "Feature",
            geometry: {
              type: "Polygon",
              coordinates: [lineCoords],
            },
          }
        : {
            type: "Feature",
            geometry: {
              type: "LineString",
              coordinates: lineCoords,
            },
          };

    if (map.getSource("draw-polygon")) {
      map.getSource("draw-polygon").setData({
        type: "FeatureCollection",
        features: [feature],
      });
    }
  } else {
    if (map.getSource("draw-polygon")) {
      map.getSource("draw-polygon").setData({
        type: "FeatureCollection",
        features: [],
      });
    }
  }
}

function handleMapClick(e) {
  if (!isDrawing.value) return;

  const coord = [e.lngLat.lng, e.lngLat.lat];
  drawPoints.push(coord);
  updateDrawLayers();

  // 3 個點以上就算面積
  if (drawPoints.length >= 3) {
    const sqMeters = calculatePolygonArea(drawPoints);
    const ping = sqMeters / 3.30579;
    areaValue.value = Math.round(ping * 10) / 10;
  }
}

function startDrawing() {
  clearDrawing();
  isDrawing.value = true;
  if (map) {
    map.getCanvas().style.cursor = "crosshair";
  }
}

function finishDrawing() {
  isDrawing.value = false;
  if (map) {
    map.getCanvas().style.cursor = "";
  }
}

function clearDrawing() {
  drawPoints = [];
  areaValue.value = "";
  isDrawing.value = false;
  updateDrawLayers();
  if (map) {
    map.getCanvas().style.cursor = "";
  }
}

// 監聽 dialog 開關
watch(
  () => dialogStore.dialogs.solarEstimate,
  async (isOpen) => {
    if (isOpen) {
      await nextTick();
      setTimeout(() => initMap(), 100);
    } else {
      destroyMap();
    }
  }
);

function initMap() {
  if (map || !mapContainer.value) return;

  mapboxgl.accessToken = import.meta.env.VITE_MAPBOXTOKEN;

  map = new mapboxgl.Map({
    container: mapContainer.value,
    style: "mapbox://styles/mapbox/satellite-v9",
    center: [121.5654, 25.033],
    zoom: 17,
  });

  map.addControl(new mapboxgl.NavigationControl(), "top-right");

  map.on("load", () => {
    // 多邊形/線 source + layer
    map.addSource("draw-polygon", {
      type: "geojson",
      data: { type: "FeatureCollection", features: [] },
    });

    map.addLayer({
      id: "draw-polygon-fill",
      type: "fill",
      source: "draw-polygon",
      paint: {
        "fill-color": "#ff9800",
        "fill-opacity": 0.3,
      },
      filter: ["==", "$type", "Polygon"],
    });

    map.addLayer({
      id: "draw-polygon-line",
      type: "line",
      source: "draw-polygon",
      paint: {
        "line-color": "#ff9800",
        "line-width": 2,
      },
    });

    // 點 source + layer
    map.addSource("draw-points", {
      type: "geojson",
      data: { type: "FeatureCollection", features: [] },
    });

    map.addLayer({
      id: "draw-points-circle",
      type: "circle",
      source: "draw-points",
      paint: {
        "circle-radius": 5,
        "circle-color": "#ff9800",
        "circle-stroke-color": "#fff",
        "circle-stroke-width": 1,
      },
    });

    // 點擊事件
    map.on("click", handleMapClick);
  });
}

function destroyMap() {
  if (map) {
    map.off("click", handleMapClick);
    map.remove();
    map = null;
  }
  drawPoints = [];
  isDrawing.value = false;
}

// 搜尋地址
function searchAddress() {
  if (!address.value || !map) return;
  const query = encodeURIComponent(address.value);
  fetch(
    `https://api.mapbox.com/geocoding/v5/mapbox.places/${query}.json?access_token=${mapboxgl.accessToken}&country=tw&limit=1`
  )
    .then((res) => res.json())
    .then((data) => {
      if (data.features && data.features.length > 0) {
        const [lng, lat] = data.features[0].center;
        map.flyTo({ center: [lng, lat], zoom: 19 });
      }
    })
    .catch(() => {});
}

// 計算結果
const estimatedCapacity = computed(() => {
  if (!areaValue.value) return 0;
  const areaNum = parseFloat(areaValue.value);
  const typeFactor =
    roofType.value === "平屋頂"
      ? 0.9
      : roofType.value === "斜屋頂"
        ? 1.0
        : roofType.value === "金屬浪板"
          ? 0.85
          : 0.8;
  const dirFactor =
    roofDirection.value === "南向"
      ? 1.0
      : roofDirection.value === "東南向" || roofDirection.value === "西南向"
        ? 0.95
        : roofDirection.value === "東向" || roofDirection.value === "西向"
          ? 0.85
          : 0.7;
  return Math.round(areaNum * typeFactor * dirFactor);
});

const annualGeneration = computed(() => {
  return Math.round(estimatedCapacity.value * 1400 * 0.8);
});

const carbonReduction = computed(() => {
  return Math.round(annualGeneration.value * 0.000509 * 100) / 100;
});

const treeEquivalent = computed(() => {
  const trees = Math.round((carbonReduction.value * 1000) / 12);
  if (trees >= 1000) return `${Math.round(trees / 1000)}K`;
  return trees.toString();
});

// API 結果
const resultData = ref(null);

async function handleEstimate() {
  if (!areaValue.value) return;
  isCalculating.value = true;
  apiError.value = false;

  const payload = {
    address: address.value,
    area: parseFloat(areaValue.value) * 3.30579,
    roof_type: roofType.value,
    roof_dir: roofDirection.value,
  };

  try {
    const response = await fetch("/api/dev/solar/estimate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (!response.ok) throw new Error("API failed");
    resultData.value = await response.json();
  } catch (err) {
    apiError.value = true;
    resultData.value = {
      estimated_capacity_kw: estimatedCapacity.value,
      annual_generation_kwh: annualGeneration.value,
      carbon_reduction_tons: carbonReduction.value,
      tree_equivalent: treeEquivalent.value,
    };
  } finally {
    isCalculating.value = false;
    showResult.value = true;
  }
}

function handleClose() {
  dialogStore.hideAllDialogs();
  setTimeout(() => {
    address.value = "";
    areaValue.value = "";
    roofType.value = "";
    roofDirection.value = "";
    showResult.value = false;
    resultData.value = null;
    apiError.value = false;
  }, 300);
}

function handleBack() {
  showResult.value = false;
  destroyMap();
  nextTick(() => {
    setTimeout(() => initMap(), 100);
  });
}

function handleConfirm() {
  handleClose();
  dialogStore.showNotification("success", "評估完成！");
}

onBeforeUnmount(() => {
  destroyMap();
});
</script>

<template>
  <DialogContainer
    dialog="solarEstimate"
    @on-close="handleClose"
  >
    <div class="solarestimate">
      <button
        class="solarestimate-close"
        @click="handleClose"
      >
        <span>close</span>
      </button>

      <!-- 表單頁 -->
      <div
        v-if="!showResult"
        class="solarestimate-form"
      >
        <h3>太陽能節碳評估</h3>

        <div class="solarestimate-field">
          <label>地址</label>
          <div class="solarestimate-input-wrap">
            <input
              v-model="address"
              type="text"
              placeholder="輸入建物地址"
              @keyup.enter="searchAddress"
            >
            <button
              class="search-btn"
              title="搜尋地址"
              @click="searchAddress"
            >
              <span>search</span>
            </button>
            <span class="info-icon">
              info
              <span class="tooltip">輸入地址後按 Enter 或點搜尋，地圖會飛到該位置</span>
            </span>
          </div>
        </div>

        <div class="solarestimate-field">
          <label>面積（坪）</label>
          <div class="solarestimate-input-wrap">
            <input
              v-model="areaValue"
              type="number"
              placeholder="在地圖畫屋頂或手動輸入"
            >
            <span class="info-icon">
              info
              <span class="tooltip">點「畫屋頂」後在地圖上逐點點擊標出範圍，自動計算面積</span>
            </span>
          </div>
        </div>

        <div class="solarestimate-field">
          <label>屋頂類型</label>
          <div class="solarestimate-input-wrap">
            <select v-model="roofType">
              <option value="" disabled>選擇屋頂類型</option>
              <option v-for="t in roofTypes" :key="t" :value="t">{{ t }}</option>
            </select>
            <span class="info-icon">
              info
              <span class="tooltip">影響安裝效率與成本</span>
            </span>
          </div>
        </div>

        <div class="solarestimate-field">
          <label>屋頂朝向</label>
          <div class="solarestimate-input-wrap">
            <select v-model="roofDirection">
              <option value="" disabled>選擇朝向</option>
              <option v-for="d in directions" :key="d" :value="d">{{ d }}</option>
            </select>
            <span class="info-icon">
              info
              <span class="tooltip">南向日照最佳</span>
            </span>
          </div>
        </div>

        <!-- 地圖區域 -->
        <div class="solarestimate-map-container">
          <div class="solarestimate-map-toolbar">
            <button
              :class="{ active: isDrawing }"
              @click="isDrawing ? finishDrawing() : startDrawing()"
            >
              <span>{{ isDrawing ? 'check' : 'edit' }}</span>
              {{ isDrawing ? '完成' : '畫屋頂' }}
            </button>
            <button @click="clearDrawing">
              <span>delete</span>
              清除
            </button>
          </div>
          <div
            ref="mapContainer"
            class="solarestimate-map"
          />
          <p class="solarestimate-map-hint">
            {{ isDrawing ? '在地圖上逐點點擊標出屋頂範圍，3 點以上自動計算面積' : '點擊「畫屋頂」開始繪製範圍' }}
          </p>
        </div>

        <button
          class="solarestimate-submit"
          :disabled="!areaValue || isCalculating"
          @click="handleEstimate"
        >
          {{ isCalculating ? "計算中..." : "評估" }}
        </button>
      </div>

      <!-- 結果頁 -->
      <div
        v-else
        class="solarestimate-result"
      >
        <button
          class="solarestimate-back"
          @click="handleBack"
        >
          <span>arrow_back</span>
          返回
        </button>

        <h3>預期一年節碳</h3>
        <p class="solarestimate-result-main">
          相當於種了
          <strong>{{ resultData?.tree_equivalent || treeEquivalent }}</strong>
          棵樹
        </p>

        <div class="solarestimate-result-trees">
          <svg
            v-for="i in 5"
            :key="i"
            viewBox="0 0 40 50"
            class="solarestimate-result-tree"
          >
            <polygon
              :points="i % 2 === 0 ? '20,2 35,30 5,30' : '20,5 33,28 7,28'"
              :fill="i % 2 === 0 ? '#6abf69' : '#8fd18e'"
            />
            <polygon
              :points="i % 2 === 0 ? '20,12 38,38 2,38' : '20,15 36,36 4,36'"
              :fill="i % 2 === 0 ? '#4a9e49' : '#6abf69'"
            />
            <rect x="17" y="36" width="6" height="10" fill="#8B7355" rx="1" />
          </svg>
        </div>

        <div v-if="apiError" class="solarestimate-result-warning">
          <span>info</span>
          <p>無法連線至評估伺服器，以下為前端預估值</p>
        </div>

        <div class="solarestimate-result-details">
          <div class="solarestimate-result-item">
            <span class="label">預估裝置容量</span>
            <span class="value">{{ Math.round((resultData?.capacity_kw || resultData?.estimated_capacity_kw || estimatedCapacity) * 10) / 10 }} kW</span>
          </div>
          <div class="solarestimate-result-item">
            <span class="label">年發電量</span>
            <span class="value">{{ Math.round(resultData?.annual_generation_kwh || annualGeneration).toLocaleString() }} 度</span>
          </div>
          <div class="solarestimate-result-item">
            <span class="label">年減碳量</span>
            <span class="value">{{ Math.round((resultData?.carbon_reduction_ton || resultData?.carbon_reduction_tons || carbonReduction) * 100) / 100 }} 公噸</span>
          </div>
          <div class="solarestimate-result-item">
            <span class="label">屋頂面積</span>
            <span class="value">{{ Math.round(areaValue * 10) / 10 }} 坪</span>
          </div>
        </div>

        <div class="solarestimate-result-actions">
          <button
            class="solarestimate-confirm"
            @click="handleConfirm"
          >
            確認
          </button>
          <a
            href="https://www.ey.gov.tw/Page/5A8A0CB5B41DA11E/00d92110-3049-44ee-9382-6988a4b2436d"
            target="_blank"
            rel="noopener noreferrer"
            class="solarestimate-learnmore"
          >
            了解更多政策
            <span>open_in_new</span>
          </a>
        </div>
      </div>
    </div>
  </DialogContainer>
</template>

<style scoped lang="scss">
.solarestimate {
  width: 400px;
  max-height: 85vh;
  overflow-y: auto;
  position: relative;
  color: var(--color-normal-text);

  &::-webkit-scrollbar {
    width: 4px !important;
  }
  &::-webkit-scrollbar-thumb {
    background: var(--color-border);
    border-radius: 2px;
  }

  &-close {
    position: absolute;
    top: 0;
    right: 0;
    z-index: 2;

    span {
      color: var(--color-complement-text);
      font-family: var(--font-icon);
      font-size: 1.2rem;
      transition: color 0.2s;

      &:hover {
        color: var(--color-normal-text);
      }
    }
  }

  &-form {
    h3 {
      font-size: 1.1rem;
      margin-bottom: 1.2rem;
    }
  }

  &-field {
    margin-bottom: 0.8rem;

    label {
      display: block;
      font-size: 0.85rem;
      font-weight: 600;
      margin-bottom: 0.3rem;
    }
  }

  &-input-wrap {
    display: flex;
    align-items: center;
    gap: 8px;

    input,
    select {
      flex: 1;
      padding: 8px 12px;
      border: 1px solid var(--color-border);
      border-radius: 5px;
      background-color: rgba(255, 255, 255, 0.08);
      color: var(--color-normal-text);
      font-size: 0.85rem;
      overflow: visible;

      &::placeholder {
        color: var(--color-complement-text);
      }

      &:focus {
        outline: none;
        border-color: var(--color-highlight);
      }
    }

    select {
      cursor: pointer;
    }

    .search-btn {
      padding: 6px;
      border: 1px solid var(--color-border);
      border-radius: 5px;
      background-color: rgba(255, 255, 255, 0.08);
      cursor: pointer;
      flex-shrink: 0;
      transition: background-color 0.2s;

      &:hover {
        background-color: rgba(255, 255, 255, 0.15);
      }

      span {
        font-family: var(--font-icon);
        font-size: 1rem;
        color: var(--color-complement-text);
      }
    }
  }

  &-map-container {
    margin: 0.8rem 0;
  }

  &-map-toolbar {
    display: flex;
    gap: 6px;
    margin-bottom: 6px;

    button {
      display: flex;
      align-items: center;
      gap: 4px;
      padding: 5px 10px;
      border: 1px solid var(--color-border);
      border-radius: 5px;
      background-color: rgba(255, 255, 255, 0.08);
      color: var(--color-complement-text);
      font-size: 0.75rem;
      cursor: pointer;
      transition: all 0.2s;

      span {
        font-family: var(--font-icon);
        font-size: 0.9rem;
      }

      &:hover {
        background-color: rgba(255, 255, 255, 0.15);
        color: var(--color-normal-text);
      }

      &.active {
        background-color: #ff9800;
        color: #222;
        border-color: #ff9800;
      }
    }
  }

  &-map {
    width: 100%;
    height: 280px;
    border-radius: 5px;
    border: 1px solid var(--color-border);
    overflow: hidden !important;

    :deep(.mapboxgl-ctrl-group) {
      background: rgba(40, 42, 44, 0.95);
      border: 1px solid var(--color-border);
      border-radius: 4px;

      button {
        border: none;
        width: 30px;
        height: 30px;

        &:hover {
          background-color: rgba(255, 255, 255, 0.15);
        }
      }
    }
  }

  &-map-hint {
    margin-top: 4px;
    font-size: 0.7rem;
    color: var(--color-complement-text);
    text-align: center;
  }

  &-submit {
    width: 100%;
    margin-top: 0.5rem;
    padding: 10px;
    border: none;
    border-radius: 5px;
    background-color: #c4a882;
    color: #333;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
    transition: background-color 0.2s;

    &:hover:not(:disabled) {
      background-color: #b89b74;
    }

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }

  &-back {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-bottom: 1rem;
    color: var(--color-complement-text);
    font-size: 0.85rem;
    transition: color 0.2s;

    span {
      font-family: var(--font-icon);
      font-size: 1rem;
    }

    &:hover {
      color: var(--color-normal-text);
    }
  }

  &-result {
    h3 {
      font-size: 1rem;
      margin-bottom: 1rem;
      color: var(--color-complement-text);
    }

    &-main {
      font-size: 1.1rem;
      margin-bottom: 1rem;

      strong {
        font-size: 2rem;
        color: #6abf69;
      }
    }

    &-trees {
      display: flex;
      justify-content: flex-end;
      gap: 4px;
      margin-bottom: 1.5rem;
    }

    &-tree {
      width: 36px;
      height: 45px;
    }

    &-warning {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 8px 12px;
      margin-bottom: 1rem;
      border-radius: 5px;
      background-color: rgba(255, 152, 0, 0.15);
      border: 1px solid rgba(255, 152, 0, 0.3);

      span {
        font-family: var(--font-icon);
        font-size: 1rem;
        color: #ff9800;
        flex-shrink: 0;
      }

      p {
        font-size: 0.75rem;
        color: #ff9800;
      }
    }

    &-details {
      border-top: 1px solid var(--color-border);
      padding-top: 1rem;
    }

    &-item {
      display: flex;
      justify-content: space-between;
      padding: 6px 0;

      .label {
        color: var(--color-complement-text);
        font-size: 0.85rem;
      }

      .value {
        font-weight: 600;
        font-size: 0.85rem;
      }
    }

    &-actions {
      display: flex;
      gap: 10px;
      margin-top: 1.2rem;
    }

    .solarestimate-confirm {
      flex: 1;
      padding: 10px;
      border: none;
      border-radius: 5px;
      background-color: #6abf69;
      color: white;
      font-size: 0.9rem;
      font-weight: 600;
      cursor: pointer;
      transition: background-color 0.2s;

      &:hover {
        background-color: #5aaf59;
      }
    }

    .solarestimate-learnmore {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 4px;
      flex: 1;
      padding: 10px;
      border: 1px solid var(--color-border);
      border-radius: 5px;
      color: var(--color-highlight);
      font-size: 0.85rem;
      text-decoration: none;
      transition: background-color 0.2s;

      span {
        font-family: var(--font-icon);
        font-size: 0.9rem;
      }

      &:hover {
        background-color: rgba(255, 255, 255, 0.05);
      }
    }
  }
}

.info-icon {
  position: relative;
  overflow: visible;
  cursor: help;
  font-family: var(--font-icon);
  font-size: 1rem;
  color: var(--color-complement-text);
  flex-shrink: 0;

  .tooltip {
    position: absolute;
    bottom: 120%;
    left: 50%;
    transform: translateX(-50%);
    background: #333;
    color: #fff;
    padding: 6px 10px;
    border-radius: 4px;
    font-size: 0.75rem;
    font-family: inherit;
    white-space: nowrap;
    opacity: 0;
    pointer-events: none;
    transition: opacity 0.2s ease;
    z-index: 10;
  }

  &:hover .tooltip {
    opacity: 1;
  }
}
</style>
