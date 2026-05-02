-- ============================================================
-- 雙北行政區年平均日照量 (district_solar_psh)
-- 資料庫：dashboard (postgres-data) + dashboardmanager (postgres-manager)
--
-- 執行方式（macOS/Linux）：
--   # Step 1：在 dashboard DB 建資料表與插入資料
--   docker cp docker/sql/add_district_solar_psh.sql postgres-data:/tmp/
--   docker exec postgres-data psql -U postgres -d dashboard \
--       -f /tmp/add_district_solar_psh.sql
--
--   # Step 2：在 dashboardmanager DB 設定組件
--   docker cp docker/sql/add_district_solar_psh.sql postgres-manager:/tmp/
--   docker exec postgres-manager psql -U postgres -d dashboardmanager \
--       -f /tmp/add_district_solar_psh.sql
--
-- 注意：兩個區段需分別對各自的資料庫執行（可拆成兩個視窗）
-- ============================================================


-- ====================
-- [1] dashboard DB
--     docker exec postgres-data psql -U postgres -d dashboard
-- ====================

-- 建立資料表
CREATE TABLE IF NOT EXISTS public.district_solar_psh (
    id       SERIAL PRIMARY KEY,
    district VARCHAR(20),
    lat      NUMERIC(10,6),
    lng      NUMERIC(10,6),
    psh      NUMERIC(6,4)
);

-- 插入 41 個雙北行政區年平均日照量資料（PSH，kWh/m²/day）
INSERT INTO public.district_solar_psh (district, lat, lng, psh) VALUES
('北投區', 25.132300, 121.500200, 3.7111),
('士林區', 25.093400, 121.524100, 3.7111),
('內湖區', 25.083000, 121.587000, 3.7111),
('南港區', 25.055000, 121.606900, 3.7111),
('松山區', 25.057500, 121.577100, 3.7111),
('信義區', 25.032400, 121.564500, 3.7111),
('中山區', 25.069700, 121.532600, 3.7111),
('大同區', 25.063000, 121.513100, 3.7111),
('中正區', 25.032000, 121.519800, 3.7111),
('萬華區', 25.034000, 121.499700, 3.7111),
('大安區', 25.026400, 121.543100, 3.7111),
('文山區', 24.998800, 121.567700, 3.8107),
('新莊區', 25.035500, 121.449900, 3.7111),
('淡水區', 25.170000, 121.443200, 3.7111),
('汐止區', 25.065000, 121.657300, 3.7111),
('板橋區', 25.012700, 121.467400, 3.7111),
('三重區', 25.061700, 121.487500, 3.7111),
('樹林區', 24.989600, 121.417800, 3.8107),
('土城區', 24.973900, 121.445000, 3.8107),
('蘆洲區', 25.085300, 121.473800, 3.7111),
('中和區', 24.998600, 121.498600, 3.8107),
('永和區', 25.014200, 121.516400, 3.7111),
('新店區', 24.957100, 121.537500, 3.8107),
('鶯歌區', 24.954100, 121.343000, 3.8107),
('三峽區', 24.934900, 121.371200, 3.8107),
('瑞芳區', 25.107600, 121.802400, 3.7111),
('五股區', 25.078000, 121.436700, 3.7111),
('泰山區', 25.054800, 121.431300, 3.7111),
('林口區', 25.077900, 121.387800, 3.7111),
('深坑區', 24.977900, 121.613800, 3.8107),
('石碇區', 24.967700, 121.659100, 3.8107),
('坪林區', 24.933300, 121.712100, 3.8107),
('三芝區', 25.257200, 121.501800, 3.7111),
('石門區', 25.293800, 121.569000, 3.7111),
('八里區', 25.152000, 121.410300, 3.7111),
('平溪區', 25.026600, 121.742500, 3.7111),
('雙溪區', 25.029600, 121.867100, 3.7111),
('貢寮區', 25.025900, 121.903700, 3.7111),
('金山區', 25.222200, 121.636700, 3.7111),
('萬里區', 25.179600, 121.685100, 3.7111),
('烏來區', 24.865900, 121.550300, 3.8107);


-- ====================
-- [2] dashboardmanager DB
--     docker exec postgres-manager psql -U postgres -d dashboardmanager
-- ====================

-- component_maps（地圖設定，fill 面圖，顏色依 PSH 日照量深淺）
INSERT INTO public.component_maps (index, title, type, source, size, icon, paint, property)
VALUES (
    'district_solar_psh',
    '雙北行政區年平均日照量',
    'fill',
    'geojson',
    NULL,
    NULL,
    '{"fill-color":["interpolate",["linear"],["get","psh"],3.7,"#FFF9C4",3.8107,"#F57C00"],"fill-opacity":0.7,"fill-outline-color":"#ffffff"}',
    '[{"key":"TNAME","name":"行政區"},{"key":"psh","name":"年平均日照(kWh/m²/day)"}]'
);

-- component_charts（圖表設定）
INSERT INTO public.component_charts (index, color, types, unit)
VALUES (
    'district_solar_psh',
    '{"#F57C00"}',
    '{"MapLegend"}',
    'kWh/m²/day'
);

-- components（組件基本資訊）
INSERT INTO public.components (index, name)
VALUES ('district_solar_psh', '雙北行政區年平均日照量');

-- query_charts（查詢設定，地圖圖例模式）
-- map_config_ids 自動查詢，無需手動填寫
INSERT INTO public.query_charts
    (index, city, query_type, query_chart, map_config_ids,
     source, short_desc, long_desc, use_case,
     time_from, update_freq, update_freq_unit,
     contributors, created_at, updated_at)
VALUES (
    'district_solar_psh', 'metrotaipei', 'map_legend',
    'SELECT district AS x_axis, psh::numeric AS data FROM public.district_solar_psh ORDER BY psh DESC',
    (SELECT ARRAY[id] FROM public.component_maps WHERE index = 'district_solar_psh'),
    '臺北市政府',
    '雙北各行政區年平均日照量（PSH），以顏色深淺顯示日照強度。',
    '本組件以面圖顯示雙北41個行政區的年平均日照量（Peak Sun Hours，kWh/m²/day）。日照量越高代表太陽能發電潛力越大，可作為推廣太陽能板設置的參考依據。',
    '可觀察雙北各行政區的太陽能發電潛力差異，輔助政策規劃與民眾評估安裝太陽能板的效益。',
    'static', 0, NULL, '{doit}', NOW(), NOW()
);

-- 將組件加入「綠能發展」儀表板（index='251c7c83aee5'）
UPDATE public.dashboards
SET components = array_append(components, (
    SELECT id FROM public.components WHERE index = 'district_solar_psh'
))
WHERE index = '251c7c83aee5'  -- 綠能發展儀表板
  AND NOT (components @> ARRAY[(
    SELECT id FROM public.components WHERE index = 'district_solar_psh'
  )]);
