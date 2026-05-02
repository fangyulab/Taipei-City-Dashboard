package ai

import (
	"TaipeiCityDashboardBE/app/models"
	"TaipeiCityDashboardBE/app/util"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

type RecommendRequest struct {
	SessionID      string
	Topic          string
	City           string
	ComponentLimit int
	BCount         int
}

type ComponentSummary struct {
	ID         int64    `json:"id"`
	Index      string   `json:"index"`
	Name       string   `json:"name"`
	City       string   `json:"city"`
	ChartTypes []string `json:"chart_types"`
	DataTypes  []string `json:"data_types"`
	ShortDesc  string   `json:"short_desc"`
	QueryType  string   `json:"query_type"`
	HasMap     bool     `json:"has_map"`
	HasHistory bool     `json:"has_history"`
}

type ComponentRecommendation struct {
	ID         int64    `json:"id"`
	Index      string   `json:"index"`
	Name       string   `json:"name"`
	City       string   `json:"city"`
	ChartTypes []string `json:"chart_types"`
	DataTypes  []string `json:"data_types"`
	Reason     string   `json:"reason"`
}

type RecommendResult struct {
	Theme              string                    `json:"theme"`
	AComponent         ComponentRecommendation   `json:"a_component"`
	BComponents        []ComponentRecommendation `json:"b_components"`
	AnalysisDirections []string                  `json:"analysis_directions"`
}

type chartConfig struct {
	Types []string `json:"types"`
}

func RecommendComponents(ctx context.Context, userID, ip string, req RecommendRequest) (*RecommendResult, string, error) {
	if req.Topic == "" {
		return nil, "", fmt.Errorf("topic is required")
	}
	if req.ComponentLimit <= 0 {
		req.ComponentLimit = 200
	}
	if req.BCount <= 0 {
		req.BCount = 4
	}
	if req.SessionID == "" {
		req.SessionID = "session_" + util.GenerateRandomString(10)
	}

	summaries, err := buildComponentSummaries(req.City, req.ComponentLimit)
	if err != nil {
		return nil, "", err
	}

	prompt, err := buildRecommendPrompt(req.Topic, req.BCount, summaries)
	if err != nil {
		return nil, "", err
	}

	systemPrompt := buildSystemPrompt()
	messages := []llms.MessageContent{
		{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextContent{Text: systemPrompt},
			},
		},
		{
			Role: llms.ChatMessageTypeHuman,
			Parts: []llms.ContentPart{
				llms.TextContent{Text: prompt},
			},
		},
	}

	aiReq := AIChatRequest{
		SessionID: req.SessionID,
		UserID:    userID,
		IPAddress: ip,
		Messages:  messages,
	}

	options := []llms.CallOption{
		llms.WithTemperature(0.3),
		llms.WithMaxTokens(900),
	}

	logEntry, err := ChatWithTWCC(ctx, aiReq, options...)
	if err != nil {
		return nil, "", err
	}

	raw := strings.TrimSpace(logEntry.Answer)
	result, err := parseRecommendResult(raw)
	if err != nil {
		return nil, raw, err
	}

	enrichRecommendations(result, summaries, req.BCount)
	return result, raw, nil
}

func buildComponentSummaries(city string, limit int) ([]ComponentSummary, error) {
	components, _, _, err := models.GetAllComponents(city, limit, 1, "", "", "", "", "", "", "")
	if err != nil {
		return nil, err
	}

	summaries := make([]ComponentSummary, 0, len(components))
	for _, c := range components {
		chartTypes := parseChartTypes(c.ChartConfig)
		hasMap := hasNonEmptyJSON(c.MapConfig)
		hasHistory := hasNonEmptyJSON(c.HistoryConfig)
		dataTypes := deriveDataTypes(chartTypes, hasMap, hasHistory)

		summaries = append(summaries, ComponentSummary{
			ID:         c.ID,
			Index:      c.Index,
			Name:       c.Name,
			City:       c.City,
			ChartTypes: chartTypes,
			DataTypes:  dataTypes,
			ShortDesc:  c.ShortDesc,
			QueryType:  c.QueryType,
			HasMap:     hasMap,
			HasHistory: hasHistory,
		})
	}

	return summaries, nil
}

func parseChartTypes(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var cfg chartConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil
	}
	return uniqueStrings(cfg.Types)
}

func hasNonEmptyJSON(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	trim := strings.TrimSpace(string(raw))
	if trim == "" || trim == "null" || trim == "[]" || trim == "{}" {
		return false
	}
	return true
}

func deriveDataTypes(chartTypes []string, hasMap bool, hasHistory bool) []string {
	typeMap := map[string][]string{
		"TimelineSeparateChart": {"time_series"},
		"TimelineStackedChart":  {"time_series"},
		"ColumnLineChart":       {"time_series", "comparison"},
		"BarChartWithGoal":      {"time_series", "comparison"},
		"BarChart":              {"comparison"},
		"ColumnChart":           {"comparison"},
		"BarPercentChart":       {"distribution"},
		"DonutChart":            {"distribution"},
		"PolarAreaChart":        {"distribution"},
		"TreemapChart":          {"composition"},
		"RadarChart":            {"profile"},
		"IndicatorChart":        {"indicator"},
		"TextUnitChart":         {"indicator"},
		"GuageChart":            {"indicator"},
		"IconPercentChart":      {"indicator"},
		"DistrictChart":         {"map"},
		"MapLegend":             {"map"},
		"MetroChart":            {"map"},
		"HeatmapChart":          {"map"},
	}

	out := make([]string, 0)
	if hasMap {
		out = append(out, "map")
	}
	if hasHistory {
		out = append(out, "history")
	}

	for _, ct := range chartTypes {
		if mapped, ok := typeMap[ct]; ok {
			out = append(out, mapped...)
		} else {
			out = append(out, "other")
		}
	}

	return uniqueStrings(out)
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func buildSystemPrompt() string {
	return strings.TrimSpace(`You are a recommendation engine for dashboard components.
Return ONLY valid JSON matching the schema. Do not include markdown or extra text.
Rules:
- Select exactly 1 A component and N B components.
- B components must have distinct data_types as much as possible.
- Use ONLY the provided components list.
- Provide analysis_directions as a list of concrete analysis angles.
Schema:
{
  "theme": "string",
  "a_component": {
    "index": "string",
    "city": "string",
    "reason": "string"
  },
  "b_components": [
    {
      "index": "string",
      "city": "string",
      "reason": "string"
    }
  ],
  "analysis_directions": ["string"]
}`)
}

func buildRecommendPrompt(topic string, bCount int, summaries []ComponentSummary) (string, error) {
	payload := map[string]interface{}{
		"topic":                 topic,
		"b_count":               bCount,
		"components":            summaries,
		"data_type_candidates": []string{"time_series", "map", "distribution", "comparison", "composition", "profile", "indicator", "history", "other"},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func parseRecommendResult(raw string) (*RecommendResult, error) {
	if raw == "" {
		return nil, fmt.Errorf("empty AI response")
	}

	jsonText := raw
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		jsonText = raw[start : end+1]
	}

	var result RecommendResult
	if err := json.Unmarshal([]byte(jsonText), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func enrichRecommendations(result *RecommendResult, summaries []ComponentSummary, targetB int) {
	if result == nil {
		return
	}

	summaryByKey := make(map[string]ComponentSummary, len(summaries))
	for _, s := range summaries {
		key := fmt.Sprintf("%s|%s", s.Index, s.City)
		summaryByKey[key] = s
	}

	result.AComponent = mergeRecommendation(result.AComponent, summaryByKey)
	usedKeys := map[string]bool{}
	if result.AComponent.Index != "" {
		usedKeys[fmt.Sprintf("%s|%s", result.AComponent.Index, result.AComponent.City)] = true
	}

	bFiltered := make([]ComponentRecommendation, 0, len(result.BComponents))
	usedTypes := map[string]bool{}

	for _, b := range result.BComponents {
		merged := mergeRecommendation(b, summaryByKey)
		key := fmt.Sprintf("%s|%s", merged.Index, merged.City)
		if key == "|" || usedKeys[key] {
			continue
		}
		if addsNewType(merged.DataTypes, usedTypes) {
			bFiltered = append(bFiltered, merged)
			usedKeys[key] = true
			markTypes(merged.DataTypes, usedTypes)
		}
	}

	if len(bFiltered) < targetB {
		for _, s := range summaries {
			if len(bFiltered) >= targetB {
				break
			}
			key := fmt.Sprintf("%s|%s", s.Index, s.City)
			if usedKeys[key] {
				continue
			}
			if !addsNewType(s.DataTypes, usedTypes) {
				continue
			}
			bFiltered = append(bFiltered, ComponentRecommendation{
				ID:         s.ID,
				Index:      s.Index,
				Name:       s.Name,
				City:       s.City,
				ChartTypes: s.ChartTypes,
				DataTypes:  s.DataTypes,
				Reason:     "Auto-selected to diversify data types.",
			})
			usedKeys[key] = true
			markTypes(s.DataTypes, usedTypes)
		}
	}

	result.BComponents = bFiltered
}

func mergeRecommendation(rec ComponentRecommendation, summaryByKey map[string]ComponentSummary) ComponentRecommendation {
	key := fmt.Sprintf("%s|%s", rec.Index, rec.City)
	if summary, ok := summaryByKey[key]; ok {
		if rec.ID == 0 {
			rec.ID = summary.ID
		}
		if rec.Name == "" {
			rec.Name = summary.Name
		}
		if len(rec.ChartTypes) == 0 {
			rec.ChartTypes = summary.ChartTypes
		}
		if len(rec.DataTypes) == 0 {
			rec.DataTypes = summary.DataTypes
		}
	}
	return rec
}

func addsNewType(types []string, used map[string]bool) bool {
	for _, t := range types {
		if !used[t] {
			return true
		}
	}
	return false
}

func markTypes(types []string, used map[string]bool) {
	for _, t := range types {
		used[t] = true
	}
}
