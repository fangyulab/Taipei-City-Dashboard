<!-- CarbonTreeChart.vue -->
<!-- 自定義圖表：顯示太陽能節碳量，等效大安森林公園數量 -->
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

const localResult = ref(null);

function checkLocalResult() {
  try {
    if (typeof localStorage === "undefined") return;

    const stored = localStorage.getItem("solarEstimateResult");
    if (stored) {
      localResult.value = JSON.parse(stored);
    }
  } catch (e) {
    // ignore
  }
}

checkLocalResult();

if (typeof window !== "undefined") {
  window.addEventListener("storage", checkLocalResult);
}

const checkInterval = setInterval(checkLocalResult, 2000);

onBeforeUnmount(() => {
  clearInterval(checkInterval);

  if (typeof window !== "undefined") {
    window.removeEventListener("storage", checkLocalResult);
  }
});

const treeCount = computed(() => {
  // 優先使用 localStorage 的即時試算結果
  if (localResult.value?.treeCount) {
    const count = localResult.value.treeCount;

    if (typeof count === "string" && count.includes("K")) {
      return parseInt(count) * 1000;
    }

    return parseInt(count) || 0;
  }

  // 接回原本後端資料：支援 props.series[0].data[0].y
  if (props.series?.[0]?.data?.[0]?.y !== undefined) {
    return props.series[0].data[0].y || 0;
  }

  // 兼容另一種格式：props.series.data[0].data[0].y
  if (props.series?.data?.[0]?.data?.[0]?.y !== undefined) {
    return props.series.data[0].data[0].y || 0;
  }

  return 0;
});

const carbonReduction = computed(() => {
  // 優先使用 localStorage 的即時試算結果
  if (localResult.value?.carbonReduction) {
    return parseFloat(localResult.value.carbonReduction) || 0;
  }

  // 接回原本後端資料：取第二筆作為減碳量
  if (props.series?.[0]?.data?.[1]?.y !== undefined) {
    return props.series[0].data[1].y || 0;
  }

  // 兼容另一種格式
  if (props.series?.data?.[0]?.data?.[1]?.y !== undefined) {
    return props.series.data[0].data[1].y || 0;
  }

  return 0;
});

const formattedParkCount = computed(() => {
  const count = treeCount.value;

  // 約 6000 棵樹 ≈ 1 座大安森林公園
  if (count >= 1000) return `${Math.round(count / 6000)}`;

  return count.toString();
});

const progress = computed(() => {
  const target = 50000;
  const pct = Math.min((treeCount.value / target) * 100, 100);
  const circumference = 2 * Math.PI * 118;

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
      <svg viewBox="0 0 300 300" class="carbontreechart-svg carbontreechart-ring-svg">
        <defs>
          <linearGradient id="daanGreenGradient" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stop-color="#d8f1d5" />
            <stop offset="55%" stop-color="#aee0a8" />
            <stop offset="100%" stop-color="#7fc97f" />
          </linearGradient>
        </defs>

        <circle
          cx="150"
          cy="150"
          r="118"
          fill="none"
          stroke="url(#daanGreenGradient)"
          stroke-width="28"
          opacity="0.35"
        />

        <circle
          cx="150"
          cy="150"
          r="118"
          fill="none"
          stroke="url(#daanGreenGradient)"
          stroke-width="28"
          stroke-linecap="round"
          :stroke-dasharray="2 * Math.PI * 118"
          :stroke-dashoffset="progress"
          transform="rotate(-90 150 150)"
          class="carbontreechart-progress"
        />
      </svg>

      <svg viewBox="0 0 300 300" class="carbontreechart-svg carbontreechart-text-svg">
        <text
          x="150"
          y="80"
          text-anchor="middle"
          dominant-baseline="middle"
          font-size="12"
          fill="var(--color-complement-text)"
        >
          <tspan>114年太陽能減碳 </tspan>
          <tspan font-size="16" font-weight="700" fill="var(--color-normal-text)">
            {{ Math.round(carbonReduction).toLocaleString() }}
          </tspan>
          <tspan> 噸</tspan>
        </text>

        <line
          x1="125"
          y1="90"
          x2="175"
          y2="90"
          stroke="var(--color-complement-text)"
          stroke-width="1"
          opacity="0.5"
        />

        <text
          x="150"
          y="110"
          text-anchor="middle"
          dominant-baseline="middle"
          font-size="16"
          fill="var(--color-complement-text)"
        >
          相當於
        </text>

        <text
          x="150"
          y="150"
          text-anchor="middle"
          dominant-baseline="middle"
          font-size="45"
          font-weight="700"
          fill="var(--color-normal-text)"
        >
          {{ formattedParkCount }}
        </text>

        <text
          x="150"
          y="190"
          text-anchor="middle"
          dominant-baseline="middle"
          font-size="16"
          fill="var(--color-complement-text)"
        >
          座大安森林公園
        </text>
      </svg>

      <div class="carbontreechart-trees">
        <svg
          width="27"
          height="34"
          viewBox="0 0 100 128"
          fill="none"
          class="carbontreechart-tree big-tree"
        >
          <path d="M22.4211 92.0139C22.4211 86.2609 27.0541 81.5972 32.7693 81.5972H39.6681C45.3832 81.5972 50.0163 86.2609 50.0163 92.0139V114.583C50.0163 120.336 45.3832 125 39.6681 125H32.7693C27.0541 125 22.4211 120.336 22.4211 114.583V92.0139Z" fill="#B38F75" />
          <path d="M27.0364 5.28991C31.831 -1.76331 42.169 -1.7633 46.9636 5.28992L71.87 41.9295C77.3527 49.995 71.6152 60.9451 61.9064 60.9451H12.0936C2.38479 60.9451 -3.3527 49.995 2.12999 41.9295L27.0364 5.28991Z" fill="#AEDBA8" />
          <path d="M27.0364 43.4844C31.831 36.4311 42.169 36.4311 46.9636 43.4844L71.87 80.1239C77.3527 88.1895 71.6152 99.1396 61.9064 99.1396H12.0936C2.38479 99.1396 -3.3527 88.1894 2.12999 80.1239L27.0364 43.4844Z" fill="#AEDBA8" />
          <path d="M63.7554 104.778C63.7554 100.728 67.011 97.4444 71.0271 97.4444H75.8749C79.8909 97.4444 83.1466 100.728 83.1466 104.778V120.667C83.1466 124.717 79.8909 128 75.8749 128H71.0271C67.011 128 63.7554 124.717 63.7554 120.667V104.778Z" fill="#B38F75" />
          <path d="M66.9986 43.7241C70.3677 38.7586 77.6323 38.7586 81.0014 43.7241L98.5033 69.5184C102.356 75.1965 98.3242 82.9054 91.5018 82.9054H56.4982C49.6758 82.9054 45.644 75.1965 49.4968 69.5183L66.9986 43.7241Z" fill="#83C07B" />
          <path d="M66.9986 70.613C70.3677 65.6475 77.6323 65.6475 81.0014 70.613L98.5033 96.4072C102.356 102.085 98.3242 109.794 91.5018 109.794H56.4982C49.6758 109.794 45.644 102.085 49.4968 96.4072L66.9986 70.613Z" fill="#83C07B" />
        </svg>
      </div>
    </div>

    <p class="carbontreechart-hint">點擊評估你的太陽能節碳潛力</p>
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
    isolation: isolate;
    width: 180px;
    height: 180px;

    @media (min-width: 1650px) {
      width: 220px;
      height: 220px;
    }
  }

  &-svg {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    overflow: visible;
    pointer-events: none;
  }

  &-ring-svg {
    z-index: 1;
  }

  &-text-svg {
    z-index: 3;
  }

  &-progress {
    transition: stroke-dashoffset 1s ease;
  }

  &-trees {
    position: absolute;
    bottom: 10px;
    right: -10px;
    z-index: 2;
    display: flex;
    align-items: flex-end;
    gap: 2px;
    pointer-events: none;
    transform: scale(1.1);
    transform-origin: right bottom;
  }

  &-tree {
    &.big-tree {
      margin-left: -6px;
      filter: drop-shadow(0 2px 3px rgba(0, 0, 0, 0.08));
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
