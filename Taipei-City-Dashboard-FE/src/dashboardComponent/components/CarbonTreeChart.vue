<!-- CarbonTreeChart.vue -->
<!-- 自定義圖表：顯示太陽能節碳量，等效種樹數量 -->
<!-- 放到：src/dashboardComponent/components/CarbonTreeChart.vue -->
<script setup>
import { computed, ref, onBeforeUnmount } from "vue";
import { useDialogStore } from "../../store/dialogStore";

const props = defineProps([
  "chart_config",
  "activeChart",
  "series",
  "map_config",
  "map_filter",
]);

const dialogStore = useDialogStore();

// 從 series 取得數據（two_d 格式）
const localResult = ref(null);

// 檢查 localStorage 有沒有評估結果
function checkLocalResult() {
  try {
    const stored = localStorage.getItem("solarEstimateResult");
    if (stored) {
      localResult.value = JSON.parse(stored);
    }
  } catch (e) {
    // ignore
  }
}

// 初始檢查 + 監聽 storage 變化
checkLocalResult();
window.addEventListener("storage", checkLocalResult);

// 定期檢查（因為同頁面 localStorage 變化不會觸發 storage event）
const checkInterval = setInterval(checkLocalResult, 2000);
onBeforeUnmount(() => clearInterval(checkInterval));

// const treeCount = computed(() => {
//   // 優先用 localStorage 的評估結果
//   if (localResult.value?.treeCount) {
//     const count = localResult.value.treeCount;
//     // 如果是 "30K" 格式，轉換成數字
//     if (typeof count === "string" && count.includes("K")) {
//       return parseInt(count) * 1000;
//     }
//     return parseInt(count) || 0;
//   }
//   // 否則用 API 資料
//   if (!props.series?.data?.[0]?.data) return 0;
//   return props.series.data[0].data[0]?.y || 0;
// });

const treeCount = computed(() => {
  if (!props.series?.[0]?.data?.[0]) return 0;
  return props.series[0].data[0].y || 0;
});

const carbonReduction = computed(() => {
  if (!props.series?.data?.[0]?.data) return 0;
  // 取第二筆資料作為減碳量（噸）
  return props.series.data[0].data[1]?.y || 0;
});

const formattedTreeCount = computed(() => {
  const count = treeCount.value;
  if (count >= 1000) return `${Math.round(count / 1000)}K`;
  return count.toString();
});

// 計算圓環進度（假設目標 50K）
const progress = computed(() => {
  const target = 50000;
  const pct = Math.min((treeCount.value / target) * 100, 100);
  const circumference = 2 * Math.PI * 90;
  return circumference - (pct / 100) * circumference;
});

function openEstimateDialog() {
  dialogStore.showDialog("solarEstimate");
}
</script>

<template>
  <div
    v-if="activeChart === 'CarbonTreeChart'"
    class="carbontreechart"
    @click="openEstimateDialog"
  >
    <div class="carbontreechart-ring">
      <svg
        viewBox="0 0 200 200"
        class="carbontreechart-svg"
      >
        <!-- 背景圓環 -->
        <circle
          cx="100"
          cy="100"
          r="90"
          fill="none"
          stroke="#2a3a2a"
          stroke-width="12"
        />
        <!-- 進度圓環 -->
        <circle
          cx="100"
          cy="100"
          r="90"
          fill="none"
          stroke="#6abf69"
          stroke-width="12"
          stroke-linecap="round"
          :stroke-dasharray="2 * Math.PI * 90"
          :stroke-dashoffset="progress"
          transform="rotate(-90 100 100)"
          class="carbontreechart-progress"
        />
      </svg>
      <div class="carbontreechart-center">
        <span class="carbontreechart-equals">= 種了</span>
        <span class="carbontreechart-number">{{ formattedTreeCount }}</span>
        <span class="carbontreechart-unit">棵樹</span>
      </div>
    </div>
    <!-- 樹的圖示 -->
    <div class="carbontreechart-trees">
      <svg
        v-for="i in 3"
        :key="i"
        viewBox="0 0 40 50"
        :class="`carbontreechart-tree tree-${i}`"
      >
        <polygon
          :points="i === 2 ? '20,2 35,30 5,30' : '20,5 33,28 7,28'"
          :fill="i === 2 ? '#6abf69' : '#8fd18e'"
        />
        <polygon
          :points="i === 2 ? '20,12 38,38 2,38' : '20,15 36,36 4,36'"
          :fill="i === 2 ? '#4a9e49' : '#6abf69'"
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
    <p class="carbontreechart-hint">
      點擊評估你的太陽能節碳潛力
    </p>
  </div>
</template>

<style scoped lang="scss">
.carbontreechart {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  position: relative;
  cursor: pointer;
  transition: transform 0.2s;

  &:hover {
    transform: scale(1.02);

    .carbontreechart-hint {
      opacity: 1;
    }
  }

  &-ring {
    position: relative;
    width: 180px;
    height: 180px;

    @media (min-width: 1650px) {
      width: 220px;
      height: 220px;
    }
  }

  &-svg {
    width: 100%;
    height: 100%;
  }

  &-progress {
    transition: stroke-dashoffset 1s ease;
  }

  &-center {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
  }

  &-equals {
    font-size: 0.75rem;
    color: var(--color-complement-text);
  }

  &-number {
    font-size: 2.5rem;
    font-weight: 700;
    color: var(--color-normal-text);
    line-height: 1;

    @media (min-width: 1650px) {
      font-size: 3rem;
    }
  }

  &-unit {
    font-size: 0.85rem;
    color: var(--color-complement-text);
  }

  &-trees {
    position: absolute;
    bottom: 25px;
    right: 20%;
    display: flex;
    align-items: flex-end;
    gap: 2px;

    @media (min-width: 1650px) {
      right: 22%;
      bottom: 35px;
    }
  }

  &-tree {
    width: 28px;
    height: 35px;
    opacity: 0.9;

    &.tree-1 {
      transform: scale(0.7);
    }
    &.tree-2 {
      transform: scale(1);
    }
    &.tree-3 {
      transform: scale(0.85);
    }
  }

  &-hint {
    position: absolute;
    bottom: 4px;
    font-size: 0.7rem;
    color: var(--color-highlight);
    opacity: 0;
    transition: opacity 0.3s;
  }
}
</style>