<!-- CarbonTreeChart.vue -->
<!-- 自定義圖表：顯示太陽能節碳量，等效大安森林公園數量 -->
<!-- 放到：src/dashboardComponent/components/CarbonTreeChart.vue -->

<script setup>
import { computed } from "vue";
import { useDialogStore } from "../../store/dialogStore";

const props = defineProps([
	"chart_config",
	"activeChart",
	"series",
	"map_config",
	"map_filter",
]);

const dialogStore = useDialogStore();

const treeCount = computed(() => {
	if (props.series?.[0]?.data?.[0]?.y !== undefined) {
		return props.series[0].data[0].y || 0;
	}
	return 0;
});

const carbonReduction = computed(() => {
	if (props.series?.[0]?.data?.[1]?.y !== undefined) {
		return props.series[0].data[1].y || 0;
	}
	return 0;
});

const formattedParkCount = computed(() => {
	const count = treeCount.value;
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

// import { computed, ref, onBeforeUnmount } from "vue";
// import { useDialogStore } from "../../store/dialogStore";

// const props = defineProps([
//   "chart_config",
//   "activeChart",
//   "series",
//   "map_config",
//   "map_filter",
// ]);

// const dialogStore = useDialogStore();

// const localResult = ref(null);

// function checkLocalResult() {
//   try {
//     if (typeof localStorage === "undefined") return;

//     const stored = localStorage.getItem("solarEstimateResult");
//     if (stored) {
//       localResult.value = JSON.parse(stored);
//     }
//   } catch (e) {
//     // ignore
//   }
// }

// checkLocalResult();

// if (typeof window !== "undefined") {
//   window.addEventListener("storage", checkLocalResult);
// }

// const checkInterval = setInterval(checkLocalResult, 2000);

// onBeforeUnmount(() => {
//   clearInterval(checkInterval);

//   if (typeof window !== "undefined") {
//     window.removeEventListener("storage", checkLocalResult);
//   }
// });

// const treeCount = computed(() => {
//   // 優先使用 localStorage 的即時試算結果
//   if (localResult.value?.treeCount) {
//     const count = localResult.value.treeCount;

//     if (typeof count === "string" && count.includes("K")) {
//       return parseInt(count) * 1000;
//     }

//     return parseInt(count) || 0;
//   }

//   // 接回原本後端資料：支援 props.series[0].data[0].y
//   if (props.series?.[0]?.data?.[0]?.y !== undefined) {
//     return props.series[0].data[0].y || 0;
//   }

//   // 兼容另一種格式：props.series.data[0].data[0].y
//   if (props.series?.data?.[0]?.data?.[0]?.y !== undefined) {
//     return props.series.data[0].data[0].y || 0;
//   }

//   return 0;
// });

// const carbonReduction = computed(() => {
//   // 優先使用 localStorage 的即時試算結果
//   if (localResult.value?.carbonReduction) {
//     return parseFloat(localResult.value.carbonReduction) || 0;
//   }

//   // 接回原本後端資料：取第二筆作為減碳量
//   if (props.series?.[0]?.data?.[1]?.y !== undefined) {
//     return props.series[0].data[1].y || 0;
//   }

//   // 兼容另一種格式
//   if (props.series?.data?.[0]?.data?.[1]?.y !== undefined) {
//     return props.series.data[0].data[1].y || 0;
//   }

//   return 0;
// });

// const formattedParkCount = computed(() => {
//   const count = treeCount.value;

//   // 約 6000 棵樹 ≈ 1 座大安森林公園
//   if (count >= 1000) return `${Math.round(count / 6000)}`;

//   return count.toString();
// });

// const progress = computed(() => {
//   const target = 50000;
//   const pct = Math.min((treeCount.value / target) * 100, 100);
//   const circumference = 2 * Math.PI * 118;

//   return circumference - (pct / 100) * circumference;
// });

// function openEstimateDialog() {
//   dialogStore.showDialog("solarEstimate");
// }
//
</script>

<template>
	<div
		v-if="activeChart === 'CarbonTreeChart'"
		class="carbontreechart"
		@click="openEstimateDialog"
	>
		<div class="carbontreechart-ring">
			<svg
				viewBox="0 0 300 300"
				class="carbontreechart-svg carbontreechart-ring-svg"
			>
				<defs>
					<linearGradient
						id="daanGreenGradient"
						x1="0%"
						y1="0%"
						x2="100%"
						y2="100%"
					>
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

			<svg
				viewBox="0 0 300 300"
				class="carbontreechart-svg carbontreechart-text-svg"
			>
				<text
					x="150"
					y="80"
					text-anchor="middle"
					dominant-baseline="middle"
					font-size="12"
					fill="var(--color-complement-text)"
				>
					<tspan>114年太陽能減碳</tspan>
					<tspan
						font-size="16"
						font-weight="700"
						fill="var(--color-normal-text)"
					>
						{{ Math.round(carbonReduction).toLocaleString() }}
					</tspan>
					<tspan>噸</tspan>
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

			<!-- 樹的圖示 -->
			<div class="carbontreechart-trees">
				<svg
					v-for="i in 3"
					:key="i"
					viewBox="0 0 40 50"
					:class="`carbontreechart-tree tree-${i}`"
				>
					<polygon
						:points="
							i === 2 ? '20,2 35,30 5,30' : '20,5 33,28 7,28'
						"
						:fill="i === 2 ? '#6abf69' : '#8fd18e'"
					/>
					<polygon
						:points="
							i === 2 ? '20,12 38,38 2,38' : '20,15 36,36 4,36'
						"
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
		transform: scale(1.3);

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
