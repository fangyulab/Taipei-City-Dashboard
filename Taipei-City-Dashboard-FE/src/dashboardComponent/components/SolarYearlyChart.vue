<!-- SolarYearlyChart.vue -->
<!-- 自定義圖表：台北/新北並排柱狀圖 + 雙北合計折線圖 -->
<!-- 放到：src/dashboardComponent/components/SolarYearlyChart.vue -->
<script setup>
import { ref, computed, watch } from "vue";
import VueApexCharts from "vue3-apexcharts";

const props = defineProps(["chart_config", "activeChart", "series"]);

// series 格式（time）：
// [{ name: "台北市", data: [{x: "2013-06-01", y: 236}, ...] },
//  { name: "新北市", data: [{x: "2013-06-01", y: 870}, ...] }]
// 或台北市版本只有一個 series

const parseSeries = computed(() => {
  if (!props.series) return [];

  // three_d 格式：series = { categories: [...], data: [{name, data: [...]}] }
  // 但傳進來的 props.series 可能是 data 陣列
  const categories = props.chart_config?.categories || [];
  const dataArr = Array.isArray(props.series) ? props.series : props.series?.data || [];

  if (dataArr.length === 0 || categories.length === 0) return [];

  const result = [];

  // 第一個 series → 柱狀（台北市）
  if (dataArr[0]) {
    result.push({
      name: dataArr[0].name || "台北市",
      type: "column",
      data: categories.map((cat, i) => ({
        x: cat,
        y: dataArr[0].data[i] || 0,
      })),
    });
  }

  // 第二個 series → 柱狀（新北市）
  if (dataArr[1]) {
    result.push({
      name: dataArr[1].name || "新北市",
      type: "column",
      data: categories.map((cat, i) => ({
        x: cat,
        y: dataArr[1].data[i] || 0,
      })),
    });
  }

  // 合計折線
  if (dataArr.length >= 2) {
    result.push({
      name: "雙北合計",
      type: "line",
      data: categories.map((cat, i) => ({
        x: cat,
        y: (dataArr[0].data[i] || 0) + (dataArr[1].data[i] || 0),
      })),
    });
  } else if (dataArr.length === 1) {
    result.push({
      name: "台北市趨勢",
      type: "line",
      data: categories.map((cat, i) => ({
        x: cat,
        y: dataArr[0].data[i] || 0,
      })),
    });
  }

  return result;
});

const chartOptions = computed(() => {
  const colors =
    props.series?.length >= 2
      ? ["#F5A524", "#5B9BD5", "#E55C5C"]
      : ["#F5A524", "#E55C5C"];

  return {
    chart: {
      toolbar: { show: false },
      stacked: false,
    },
    colors: colors,
    dataLabels: { enabled: false },
    grid: { show: false },
    legend: {
      show: true,
      position: "bottom",
      markers: {
        radius: [0, 0, 20],
      },
    },
    markers: {
      hover: { size: 5 },
      shape: "circle",
      size: [0, 0, 4],
      strokeWidth: 0,
    },
    stroke: {
      curve: "smooth",
      width: [0, 0, 3],
    },
    plotOptions: {
      bar: {
        columnWidth: "60%",
      },
    },
    tooltip: {
      custom: function ({ series, seriesIndex, dataPointIndex, w }) {
        const year = w.config.series[0].data[dataPointIndex]?.x || "";
        const name = w.globals.seriesNames[seriesIndex];
        const val = series[seriesIndex][dataPointIndex];
        return (
          '<div class="chart-tooltip">' +
          "<h6>" + year + " - " + name + "</h6>" +
          "<span>" + (val || 0).toLocaleString() + " " + (props.chart_config?.unit || "瓩") + "</span>" +
          "</div>"
        );
      },
      enabled: true,
      followCursor: true,
      shared: false,
    },
    xaxis: {
      type: "category",
      axisBorder: { color: "#555", height: "0.8" },
      axisTicks: { show: false },
      labels: {
        style: { colors: "var(--color-complement-text)" },
      },
    },
    yaxis: {
      min: 0,
      labels: {
        formatter: function (val) {
          if (val >= 1000) return Math.round(val / 1000) + "K";
          return Math.round(val);
        },
        style: { colors: "var(--color-complement-text)" },
      },
      title: {
        text: props.chart_config?.unit || "瓩",
        style: { color: "var(--color-complement-text)" },
      },
    },
  };
});
</script>

<template>
  <div v-if="activeChart === 'SolarYearlyChart'">
    <VueApexCharts
      type="line"
      width="100%"
      height="260px"
      :options="chartOptions"
      :series="parseSeries"
    />
  </div>
</template>