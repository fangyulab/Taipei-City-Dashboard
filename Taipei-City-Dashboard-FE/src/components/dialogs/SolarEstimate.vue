<!-- SolarEstimate.vue -->
<!-- 自定義彈跳視窗：太陽能評估表單 + 地圖畫屋頂 + 結果頁 -->
<!-- 放到：src/components/dialogs/SolarEstimate.vue -->
<script setup>
import { ref, computed, onBeforeUnmount, nextTick, watch } from "vue";
import { useDialogStore } from "../../store/dialogStore";
import DialogContainer from "./DialogContainer.vue";
import mapboxgl from "mapbox-gl";
import MapboxDraw from "@mapbox/mapbox-gl-draw";
import "@mapbox/mapbox-gl-draw/dist/mapbox-gl-draw.css";
import area from "@turf/area";

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
let draw = null;

// 屋頂類型選項
const roofTypes = ["平屋頂", "斜屋頂", "金屬浪板", "其他"];
const directions = ["南向", "東南向", "西南向", "東向", "西向", "北向"];

// 監聽 dialog 開關，開啟時初始化地圖
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
    center: [121.5654, 25.0330], // 台北市中心
    zoom: 17,
  });

  draw = new MapboxDraw({
    displayControlsDefault: false,
    controls: {
      polygon: true,
      trash: true,
    },
    defaultMode: "draw_polygon",
    styles: [
      // 多邊形填色
      {
        id: "gl-draw-polygon-fill",
        type: "fill",
        filter: ["all", ["==", "$type", "Polygon"]],
        paint: {
          "fill-color": "#ff9800",
          "fill-opacity": 0.3,
        },
      },
      // 多邊形邊框
      {
        id: "gl-draw-polygon-stroke",
        type: "line",
        filter: ["all", ["==", "$type", "Polygon"]],
        paint: {
          "line-color": "#ff9800",
          "line-width": 2,
        },
      },
      // 頂點
      {
        id: "gl-draw-point",
        type: "circle",
        filter: ["all", ["==", "$type", "Point"], ["==", "meta", "vertex"]],
        paint: {
          "circle-radius": 5,
          "circle-color": "#ff9800",
        },
      },
      // 中間點
      {
        id: "gl-draw-point-mid",
        type: "circle",
        filter: ["all", ["==", "$type", "Point"], ["==", "meta", "midpoint"]],
        paint: {
          "circle-radius": 3,
          "circle-color": "#ff9800",
        },
      },
    ],
  });

  map.addControl(draw);
  map.addControl(new mapboxgl.NavigationControl(), "top-right");

  // 畫完多邊形時計算面積
  map.on("draw.create", updateArea);
  map.on("draw.update", updateArea);
  map.on("draw.delete", () => {
    areaValue.value = "";
  });
}

function updateArea() {
  const data = draw.getAll();
  if (data.features.length > 0) {
    // 只取最後一個多邊形
    const lastFeature = data.features[data.features.length - 1];
    // 計算面積（平方公尺）
    const sqMeters = area(lastFeature);
    // 轉換為坪（1坪 = 3.30579 平方公尺）
    const ping = sqMeters / 3.30579;
    areaValue.value = Math.round(ping * 10) / 10;

    // 刪除之前的多邊形，只保留最新的
    if (data.features.length > 1) {
      const idsToRemove = data.features
        .slice(0, -1)
        .map((f) => f.id);
      idsToRemove.forEach((id) => draw.delete(id));
    }
  }
}

function destroyMap() {
  if (map) {
    map.remove();
    map = null;
    draw = null;
  }
}

// 搜尋地址並飛到該位置
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

// API 結果（從後端或預設）
const resultData = ref(null);

async function handleEstimate() {
  if (!areaValue.value) return;
  isCalculating.value = true;
  apiError.value = false;

  const payload = {
    address: address.value,
    area_ping: parseFloat(areaValue.value),
    roof_type: roofType.value,
    roof_direction: roofDirection.value,
  };

  try {
    // 嘗試呼叫後端 API
    const response = await fetch(
      "http://localhost:8080/api/v1/solar/estimate",
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      }
    );

    if (!response.ok) throw new Error("API failed");
    resultData.value = await response.json();
  } catch (err) {
    // API 失敗，使用前端預設計算
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
  // 重新初始化地圖
  destroyMap();
  nextTick(() => {
    setTimeout(() => initMap(), 100);
  });
}

function handleConfirm() {
  // 透過 event 通知外部更新樹數量
  const trees = resultData.value?.tree_equivalent || treeEquivalent.value;
  // 存到 localStorage 讓 CarbonTreeChart 讀取
  localStorage.setItem("solarEstimateResult", JSON.stringify({
    treeCount: trees,
    carbonReduction: resultData.value?.carbon_reduction_tons || carbonReduction.value,
    timestamp: Date.now(),
  }));
  handleClose();
  dialogStore.showNotification("success", `評估完成！預計等效種植 ${trees} 棵樹`);
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
      <!-- 關閉按鈕 -->
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
            <span
              class="info-icon"
              title="輸入地址後按 Enter 或點搜尋，地圖會飛到該位置"
            >info</span>
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
            <span
              class="info-icon"
              title="在地圖上畫出屋頂範圍會自動計算面積，也可手動輸入"
            >info</span>
          </div>
        </div>

        <div class="solarestimate-field">
          <label>屋頂類型</label>
          <div class="solarestimate-input-wrap">
            <select v-model="roofType">
              <option
                value=""
                disabled
              >
                選擇屋頂類型
              </option>
              <option
                v-for="t in roofTypes"
                :key="t"
                :value="t"
              >
                {{ t }}
              </option>
            </select>
            <span
              class="info-icon"
              title="影響安裝效率與成本"
            >info</span>
          </div>
        </div>

        <div class="solarestimate-field">
          <label>屋頂朝向</label>
          <div class="solarestimate-input-wrap">
            <select v-model="roofDirection">
              <option
                value=""
                disabled
              >
                選擇朝向
              </option>
              <option
                v-for="d in directions"
                :key="d"
                :value="d"
              >
                {{ d }}
              </option>
            </select>
            <span
              class="info-icon"
              title="南向日照最佳"
            >info</span>
          </div>
        </div>

        <!-- 地圖區域 -->
        <div class="solarestimate-map-container">
          <div
            ref="mapContainer"
            class="solarestimate-map"
          />
          <p class="solarestimate-map-hint">
            在衛星圖上畫出屋頂範圍，自動計算面積
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
          <strong>{{
            resultData?.tree_equivalent || treeEquivalent
          }}</strong>
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
              :points="
                i % 2 === 0 ? '20,2 35,30 5,30' : '20,5 33,28 7,28'
              "
              :fill="i % 2 === 0 ? '#6abf69' : '#8fd18e'"
            />
            <polygon
              :points="
                i % 2 === 0 ? '20,12 38,38 2,38' : '20,15 36,36 4,36'
              "
              :fill="i % 2 === 0 ? '#4a9e49' : '#6abf69'"
            />
            <rect
              x="17"
              y="36"
              width="6"
              height="10"
              fill="#8B7355"
              rx="1"
            />
          </svg>
        </div>

        <div
          v-if="apiError"
          class="solarestimate-result-warning"
        >
          <span>info</span>
          <p>無法連線至評估伺服器，以下為前端預估值</p>
        </div>

        <!-- <div class="solarestimate-result-details">
          <div class="solarestimate-result-item">
            <span class="label">預估裝置容量</span>
            <span class="value">{{
              resultData?.estimated_capacity_kw || estimatedCapacity
            }}
              kW</span>
          </div>
          <div class="solarestimate-result-item">
            <span class="label">年發電量</span>
            <span class="value">{{
              (
                resultData?.annual_generation_kwh || annualGeneration
              ).toLocaleString()
            }}
              度</span>
          </div>
          <div class="solarestimate-result-item">
            <span class="label">年減碳量</span>
            <span class="value">{{
              resultData?.carbon_reduction_tons || carbonReduction
            }}
              公噸</span>
          </div>
          <div class="solarestimate-result-item">
            <span class="label">屋頂面積</span>
            <span class="value">{{ areaValue }} 坪</span>
          </div>
        </div> -->
      <!-- </div> -->
<div class="solarestimate-result-details">
          <div class="solarestimate-result-item">
            <span class="label">預估裝置容量</span>
            <span class="value">{{
              resultData?.estimated_capacity_kw || estimatedCapacity
            }}
              kW</span>
          </div>
          <div class="solarestimate-result-item">
            <span class="label">年發電量</span>
            <span class="value">{{
              (
                resultData?.annual_generation_kwh || annualGeneration
              ).toLocaleString()
            }}
              度</span>
          </div>
          <div class="solarestimate-result-item">
            <span class="label">年減碳量</span>
            <span class="value">{{
              resultData?.carbon_reduction_tons || carbonReduction
            }}
              公噸</span>
          </div>
          <div class="solarestimate-result-item">
            <span class="label">屋頂面積</span>
            <span class="value">{{ areaValue }} 坪</span>
          </div>
        </div>

        <div class="solarestimate-result-actions">
          <button
            class="solarestimate-confirm"
            @click="handleConfirm"
          >
            確認
          </button>
          
          <a href="https://www.ey.gov.tw/Page/5A8A0CB5B41DA11E/00d92110-3049-44ee-9382-6988a4b2436d"
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

    .info-icon {
      font-family: var(--font-icon);
      font-size: 1rem;
      color: var(--color-complement-text);
      cursor: help;
      flex-shrink: 0;
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

  &-map {
    width: 100%;
    height: 280px;
    border-radius: 5px;
    border: 1px solid var(--color-border);
    overflow: hidden !important;

    :deep(.mapbox-gl-draw_ctrl-draw-btn) {
      background-color: rgba(50, 50, 50, 0.9) !important;
      border: 1px solid #666 !important;
      filter: invert(1);
      width: 30px;
      height: 30px;

      &:hover {
        background-color: rgba(80, 80, 80, 0.9) !important;
      }
    }

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

    :deep(.mapboxgl-ctrl-group:not(:empty)) {
      box-shadow: 0 0 4px rgba(0, 0, 0, 0.3);
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
</style>