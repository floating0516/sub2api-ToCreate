package service

import (
	"sort"
	"strings"
)

// channelMonitorV2CollapseDisplayGroups merges raw monitor groups into the
// three product channels shown on /monitor. Set to false to show every group.
const channelMonitorV2CollapseDisplayGroups = true

type channelMonitorV2DisplayGroup struct {
	Key      string
	Name     string
	GroupIDs []int64
}

// Edit this list to change which raw groups appear as a single matrix row.
var channelMonitorV2DisplayGroups = []channelMonitorV2DisplayGroup{
	{Key: "pro", Name: "Pro 渠道", GroupIDs: []int64{15, 18, 21, 27, 29, 30, 31}},
	{Key: "claude_opus", Name: "Claude Opus", GroupIDs: []int64{12, 16, 35, 37}},
	{Key: "claude_06", Name: "Claude 0.6", GroupIDs: []int64{2}},
}

func resolveChannelMonitorV2DisplayGroup(groupID int64, name, platform string) *channelMonitorV2DisplayGroup {
	if !channelMonitorV2CollapseDisplayGroups {
		return nil
	}
	for i := range channelMonitorV2DisplayGroups {
		for _, id := range channelMonitorV2DisplayGroups[i].GroupIDs {
			if groupID > 0 && id == groupID {
				return &channelMonitorV2DisplayGroups[i]
			}
		}
	}
	lowerName := strings.ToLower(strings.TrimSpace(name))
	lowerPlatform := strings.ToLower(strings.TrimSpace(platform))
	if strings.Contains(lowerName, "0.6") || strings.Contains(lowerName, "claude_0.6") {
		return displayGroupByKey("claude_06")
	}
	if strings.Contains(lowerName, "公益") || strings.Contains(lowerName, "public") || strings.Contains(lowerName, "free") {
		return nil
	}
	if lowerPlatform == "openai" || strings.Contains(lowerName, "gpt") || strings.Contains(lowerName, "pro") || strings.Contains(lowerName, "拼车") {
		return displayGroupByKey("pro")
	}
	if lowerPlatform == "anthropic" || strings.Contains(lowerName, "claude") || strings.Contains(lowerName, "opus") {
		return displayGroupByKey("claude_opus")
	}
	return nil
}

func displayGroupByKey(key string) *channelMonitorV2DisplayGroup {
	for i := range channelMonitorV2DisplayGroups {
		if channelMonitorV2DisplayGroups[i].Key == key {
			return &channelMonitorV2DisplayGroups[i]
		}
	}
	return nil
}

// CollapseChannelMonitorV2DisplayGroups rewrites matrix rows so /monitor shows
// the configured product channels instead of every raw group. Call before
// user-facing redaction so success_rate can be weighted by request_count.
func CollapseChannelMonitorV2DisplayGroups(matrix *ChannelMonitorV2Matrix, groupBy ChannelMonitorV2GroupBy) {
	if matrix == nil || !channelMonitorV2CollapseDisplayGroups {
		return
	}
	if groupBy != ChannelMonitorV2GroupByPlatformGroup && groupBy != ChannelMonitorV2GroupByPlatformGroupModel {
		return
	}

	type bucket struct {
		group channelMonitorV2DisplayGroup
		model string
		rows  []ChannelMonitorV2MatrixRow
	}
	buckets := map[string]*bucket{}
	order := make([]string, 0)
	for _, row := range matrix.Items {
		var groupID int64
		if row.GroupID != nil {
			groupID = *row.GroupID
		}
		group := resolveChannelMonitorV2DisplayGroup(groupID, row.GroupName, row.Platform)
		if group == nil {
			continue
		}
		key := group.Key
		if groupBy == ChannelMonitorV2GroupByPlatformGroupModel {
			key = group.Key + "\x00" + row.Model
		}
		item, ok := buckets[key]
		if !ok {
			copied := *group
			item = &bucket{group: copied, model: row.Model}
			buckets[key] = item
			order = append(order, key)
		}
		id := copiedFirstGroupID(group)
		row.Platform = ""
		row.GroupID = &id
		row.GroupName = group.Name
		item.rows = append(item.rows, row)
	}

	out := make([]ChannelMonitorV2MatrixRow, 0, len(channelMonitorV2DisplayGroups))
	for _, group := range channelMonitorV2DisplayGroups {
		if groupBy == ChannelMonitorV2GroupByPlatformGroup {
			item := buckets[group.Key]
			if item == nil {
				continue
			}
			out = append(out, mergeChannelMonitorV2MatrixRows(item.rows))
			continue
		}
		var modelKeys []string
		for _, key := range order {
			if strings.HasPrefix(key, group.Key+"\x00") {
				modelKeys = append(modelKeys, key)
			}
		}
		sort.Strings(modelKeys)
		for _, key := range modelKeys {
			out = append(out, mergeChannelMonitorV2MatrixRows(buckets[key].rows))
		}
	}
	matrix.Items = out
}

func copiedFirstGroupID(group *channelMonitorV2DisplayGroup) int64 {
	if group == nil || len(group.GroupIDs) == 0 {
		return 0
	}
	return group.GroupIDs[0]
}

func mergeChannelMonitorV2MatrixRows(rows []ChannelMonitorV2MatrixRow) ChannelMonitorV2MatrixRow {
	out := rows[0]
	if len(rows) == 1 {
		return out
	}
	metrics := make([]ChannelMonitorV2Metric, 0, len(rows))
	healths := make([]ChannelMonitorV2Health, 0, len(rows))
	bucketMap := map[int64][]ChannelMonitorV2TrendPoint{}
	for _, row := range rows {
		metrics = append(metrics, row.Metrics)
		healths = append(healths, row.Health)
		for _, bucket := range row.Buckets {
			key := bucket.BucketStart.Unix()
			bucketMap[key] = append(bucketMap[key], bucket)
		}
	}
	out.Metrics = mergeChannelMonitorV2Metrics(metrics)
	if len(healths) > 0 && healths[0].Thresholds.MinimumSample > 0 {
		out.Health = ChannelMonitorV2HealthForWithThresholds(out.Metrics, healths[0].Thresholds)
	} else {
		out.Health = ChannelMonitorV2HealthFor(out.Metrics)
	}

	starts := make([]int64, 0, len(bucketMap))
	for key := range bucketMap {
		starts = append(starts, key)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	out.Buckets = make([]ChannelMonitorV2TrendPoint, 0, len(starts))
	for _, key := range starts {
		list := bucketMap[key]
		point := list[0]
		pointMetrics := make([]ChannelMonitorV2Metric, 0, len(list))
		for _, item := range list {
			pointMetrics = append(pointMetrics, item.Metrics)
		}
		point.Metrics = mergeChannelMonitorV2Metrics(pointMetrics)
		if len(healths) > 0 && healths[0].Thresholds.MinimumSample > 0 {
			point.Health = ChannelMonitorV2HealthForWithThresholds(point.Metrics, healths[0].Thresholds)
		} else {
			point.Health = ChannelMonitorV2HealthFor(point.Metrics)
		}
		out.Buckets = append(out.Buckets, point)
	}
	return out
}

func mergeChannelMonitorV2Metrics(list []ChannelMonitorV2Metric) ChannelMonitorV2Metric {
	var out ChannelMonitorV2Metric
	var scoredError float64
	var weight int64
	var ttftP50, ttftP90, ttftP95, durP50, durP90, durP95 int64
	var ttftAvg, durAvg float64
	var ttftP50W, ttftP90W, ttftP95W, durP50W, durP90W, durP95W, ttftAvgW, durAvgW int64
	for _, item := range list {
		out.SuccessRequests += item.SuccessRequests
		out.ErrorRequests += item.ErrorRequests
		out.RequestCount += item.RequestCount
		out.InputTokens += item.InputTokens
		out.OutputTokens += item.OutputTokens
		out.CacheCreationTokens += item.CacheCreationTokens
		out.CacheReadTokens += item.CacheReadTokens
		out.TokenCount += item.TokenCount
		out.CacheRateNumerator += item.CacheRateNumerator
		out.CacheRateDenominator += item.CacheRateDenominator
		out.RPM += item.RPM
		out.TPM += item.TPM
		w := item.RequestCount
		if w <= 0 {
			w = 1
		}
		scoredError += item.ErrorRate * float64(w)
		weight += w
		out.TTFT.SampleCount += item.TTFT.SampleCount
		out.Duration.SampleCount += item.Duration.SampleCount
		addWeightedInt(&ttftP50, &ttftP50W, item.TTFT.P50Ms, w)
		addWeightedInt(&ttftP90, &ttftP90W, item.TTFT.P90Ms, w)
		addWeightedInt(&ttftP95, &ttftP95W, item.TTFT.P95Ms, w)
		addWeightedFloat(&ttftAvg, &ttftAvgW, item.TTFT.AvgMs, w)
		addWeightedInt(&durP50, &durP50W, item.Duration.P50Ms, w)
		addWeightedInt(&durP90, &durP90W, item.Duration.P90Ms, w)
		addWeightedInt(&durP95, &durP95W, item.Duration.P95Ms, w)
		addWeightedFloat(&durAvg, &durAvgW, item.Duration.AvgMs, w)
		if item.UpstreamAffectedRequests != nil {
			value := int64(0)
			if out.UpstreamAffectedRequests != nil {
				value = *out.UpstreamAffectedRequests
			}
			value += *item.UpstreamAffectedRequests
			out.UpstreamAffectedRequests = &value
		}
		if item.UpstreamAttemptCount != nil {
			value := int64(0)
			if out.UpstreamAttemptCount != nil {
				value = *out.UpstreamAttemptCount
			}
			value += *item.UpstreamAttemptCount
			out.UpstreamAttemptCount = &value
		}
	}
	if out.RequestCount > 0 {
		out.SuccessRate = float64(out.SuccessRequests) / float64(out.RequestCount)
	}
	if weight > 0 {
		out.ErrorRate = scoredError / float64(weight)
	}
	if out.CacheRateDenominator > 0 {
		out.CacheRate = float64(out.CacheRateNumerator) / float64(out.CacheRateDenominator)
	}
	out.TTFT.P50Ms = maybeInt(ttftP50, ttftP50W)
	out.TTFT.P90Ms = maybeInt(ttftP90, ttftP90W)
	out.TTFT.P95Ms = maybeInt(ttftP95, ttftP95W)
	out.TTFT.AvgMs = maybeFloat(ttftAvg, ttftAvgW)
	out.Duration.P50Ms = maybeInt(durP50, durP50W)
	out.Duration.P90Ms = maybeInt(durP90, durP90W)
	out.Duration.P95Ms = maybeInt(durP95, durP95W)
	out.Duration.AvgMs = maybeFloat(durAvg, durAvgW)
	return out
}

func addWeightedInt(sum *int64, weight *int64, value *int64, w int64) {
	if value == nil || w <= 0 {
		return
	}
	*sum += *value * w
	*weight += w
}

func addWeightedFloat(sum *float64, weight *int64, value *float64, w int64) {
	if value == nil || w <= 0 {
		return
	}
	*sum += *value * float64(w)
	*weight += w
}

func maybeInt(sum, weight int64) *int64 {
	if weight <= 0 {
		return nil
	}
	value := sum / weight
	return &value
}

func maybeFloat(sum float64, weight int64) *float64 {
	if weight <= 0 {
		return nil
	}
	value := sum / float64(weight)
	return &value
}
