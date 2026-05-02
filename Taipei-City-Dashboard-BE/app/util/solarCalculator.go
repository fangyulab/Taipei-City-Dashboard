package util

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// 行政區經緯度
var DistrictCoords = map[string][2]float64{
	// 台北市
	"北投區": {25.1323, 121.5002},
	"士林區": {25.0934, 121.5241},
	"內湖區": {25.0830, 121.5870},
	"南港區": {25.0550, 121.6069},
	"松山區": {25.0575, 121.5771},
	"信義區": {25.0324, 121.5645},
	"中山區": {25.0697, 121.5326},
	"大同區": {25.0630, 121.5131},
	"中正區": {25.0320, 121.5198},
	"萬華區": {25.0340, 121.4997},
	"大安區": {25.0264, 121.5431},
	"文山區": {24.9988, 121.5677},
	// 新北市
	"新莊區": {25.0355, 121.4499},
	"淡水區": {25.1700, 121.4432},
	"汐止區": {25.0650, 121.6573},
	"板橋區": {25.0127, 121.4674},
	"三重區": {25.0617, 121.4875},
	"樹林區": {24.9896, 121.4178},
	"土城區": {24.9739, 121.4450},
	"蘆洲區": {25.0853, 121.4738},
	"中和區": {24.9986, 121.4986},
	"永和區": {25.0142, 121.5164},
	"新店區": {24.9571, 121.5375},
	"鶯歌區": {24.9541, 121.3430},
	"三峽區": {24.9349, 121.3712},
	"瑞芳區": {25.1076, 121.8024},
	"五股區": {25.0780, 121.4367},
	"泰山區": {25.0548, 121.4313},
	"林口區": {25.0779, 121.3878},
	"深坑區": {24.9779, 121.6138},
	"石碇區": {24.9677, 121.6591},
	"坪林區": {24.9333, 121.7121},
	"三芝區": {25.2572, 121.5018},
	"石門區": {25.2938, 121.5690},
	"八里區": {25.1520, 121.4103},
	"平溪區": {25.0266, 121.7425},
	"雙溪區": {25.0296, 121.8671},
	"貢寮區": {25.0259, 121.9037},
	"金山區": {25.2222, 121.6367},
	"萬里區": {25.1796, 121.6851},
	"烏來區": {24.8659, 121.5503},
}

// 計算參數
var DirectionFactor = map[string]float64{
	"南": 1.0, "東南": 0.95, "西南": 0.95,
	"東": 0.85, "西": 0.85,
	"東北": 0.70, "西北": 0.70, "北": 0.60,
}

var CoverageRate = map[string]float64{
	"平屋頂":  0.70,
	"斜屋頂":  0.50,
	"鐵皮屋頂": 0.65,
}

const (
	EfficiencyPercent = 18.0
	CO2PerKwh         = 0.00045
	CO2PerTree        = 0.012
)

// Input / Output 型別
type SolarInput struct {
	Address  string  `json:"address"`
	Area     float64 `json:"area"`
	RoofType string  `json:"roof_type"`
	RoofDir  string  `json:"roof_dir"`
}

type SolarResult struct {
	CapacityKw          float64 `json:"capacity_kw"`
	AnnualGenerationKwh float64 `json:"annual_generation_kwh"`
	CarbonReductionTon  float64 `json:"carbon_reduction_ton"`
	TreeEquivalent      int     `json:"tree_equivalent"`
}

// 計算函式
func CalcSolarCarbon(input SolarInput) (SolarResult, error) {
	coverage := CoverageRate[input.RoofType]
	if coverage == 0 {
		coverage = 0.6
	}

	dirFactor := DirectionFactor[input.RoofDir]
	if dirFactor == 0 {
		dirFactor = 0.85
	}

	psh := GetDistrictPSH(input.Address)

	capacityKw := input.Area * coverage * (EfficiencyPercent / 100) * dirFactor
	annualKwh := capacityKw * psh * 365
	carbonTon := annualKwh * CO2PerKwh
	trees := int(carbonTon / CO2PerTree)

	return SolarResult{
		CapacityKw:          capacityKw,
		AnnualGenerationKwh: annualKwh,
		CarbonReductionTon:  carbonTon,
		TreeEquivalent:      trees,
	}, nil
}

// NASA PSH 快取
var DistrictPSH = map[string]float64{}
var pshMu sync.RWMutex

type nasaResponse struct {
	Properties struct {
		Parameter struct {
			ALLSKY_SFC_SW_DWN map[string]float64 `json:"ALLSKY_SFC_SW_DWN"`
		} `json:"parameter"`
	} `json:"properties"`
}

func fetchPSH(district string, lat, lon float64) (float64, error) {
	url := fmt.Sprintf(
		"https://power.larc.nasa.gov/api/temporal/climatology/point?parameters=ALLSKY_SFC_SW_DWN&community=RE&longitude=%.4f&latitude=%.4f&format=JSON",
		lon, lat,
	)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("NASA API 請求失敗 (%s): %w", district, err)
	}
	defer resp.Body.Close()

	var result nasaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("NASA API 解析失敗 (%s): %w", district, err)
	}

	ann, ok := result.Properties.Parameter.ALLSKY_SFC_SW_DWN["ANN"]
	if !ok {
		return 0, fmt.Errorf("NASA API 無年均值 (%s)", district)
	}

	return ann, nil
}

func fetchAllDistricts() {
	log.Println("[Solar] 開始從 NASA POWER API 載入各行政區日照資料...")

	var wg sync.WaitGroup
	var mu sync.Mutex
	newData := map[string]float64{}

	for district, coords := range DistrictCoords {
		wg.Add(1)
		go func(d string, c [2]float64) {
			defer wg.Done()

			psh, err := fetchPSH(d, c[0], c[1])
			if err != nil {
				log.Printf("[Solar] ⚠️  %s 查詢失敗，保留舊值：%v", d, err)
				pshMu.RLock()
				if old, ok := DistrictPSH[d]; ok {
					psh = old
				} else {
					psh = 2.8
				}
				pshMu.RUnlock()
			} else {
				log.Printf("[Solar] ✅  %s PSH = %.2f kWh/m²/day", d, psh)
			}

			mu.Lock()
			newData[d] = psh
			mu.Unlock()
		}(district, coords)
	}

	wg.Wait()

	pshMu.Lock()
	for k, v := range newData {
		DistrictPSH[k] = v
	}
	pshMu.Unlock()

	log.Println("[Solar] 日照資料載入完成")
}

func InitDistrictPSH() {
	fetchAllDistricts()

	go func() {
		ticker := time.NewTicker(7 * 24 * time.Hour)
		defer ticker.Stop()
		for {
			<-ticker.C
			log.Println("[Solar] 每週定期更新日照資料...")
			fetchAllDistricts()
		}
	}()
}

func GetDistrictPSH(district string) float64 {
	pshMu.RLock()
	defer pshMu.RUnlock()

	if val, ok := DistrictPSH[district]; ok {
		return val
	}
	return 2.8
}