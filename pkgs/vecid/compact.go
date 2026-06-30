package vecid

import "math"

type compactSelection struct {
	indices []int
}

type compactCluster struct {
	label    int
	members  []int
	centroid []float32
	recent   int
}

type scoredMember struct {
	idx int
	sim float32
}

func compactEmbeddings(normed [][]float32, labels []int, maxSamples int) compactSelection {
	if maxSamples <= 0 || len(normed) <= maxSamples {
		indices := make([]int, len(normed))
		for i := range normed {
			indices[i] = i
		}
		return compactSelection{indices: indices}
	}

	clustersByLabel := make(map[int]*compactCluster)
	var noise []int
	dim := 0
	if len(normed) > 0 {
		dim = len(normed[0])
	}

	for i, label := range labels {
		if label <= 0 {
			noise = append(noise, i)
			continue
		}
		c := clustersByLabel[label]
		if c == nil {
			c = &compactCluster{
				label:    label,
				centroid: make([]float32, dim),
				recent:   i,
			}
			clustersByLabel[label] = c
		}
		c.members = append(c.members, i)
		if i > c.recent {
			c.recent = i
		}
		for d := range c.centroid {
			c.centroid[d] += normed[i][d]
		}
	}

	clusters := make([]*compactCluster, 0, len(clustersByLabel))
	for _, c := range clustersByLabel {
		for d := range c.centroid {
			c.centroid[d] /= float32(len(c.members))
		}
		l2Norm(c.centroid)
		clusters = append(clusters, c)
	}
	sortClustersForCompaction(clusters)

	quotas := make(map[int]int, len(clusters))
	remaining := maxSamples
	for _, c := range clusters {
		if remaining == 0 {
			break
		}
		quotas[c.label] = 1
		remaining--
	}
	for remaining > 0 {
		bestLabel := 0
		bestScore := -1.0
		for _, c := range clusters {
			quota := quotas[c.label]
			if quota == 0 || quota >= len(c.members) {
				continue
			}
			score := math.Sqrt(float64(len(c.members))) / float64(quota+1)
			if score > bestScore {
				bestScore = score
				bestLabel = c.label
			}
		}
		if bestLabel == 0 {
			break
		}
		quotas[bestLabel]++
		remaining--
	}

	keep := make(map[int]struct{}, maxSamples)
	for _, c := range clusters {
		quota := quotas[c.label]
		if quota == 0 {
			continue
		}
		for _, idx := range selectClusterRepresentatives(normed, c.members, c.centroid, quota) {
			keep[idx] = struct{}{}
		}
	}

	if remaining > 0 && len(noise) > 0 {
		for i := len(noise) - 1; i >= 0 && remaining > 0; i-- {
			keep[noise[i]] = struct{}{}
			remaining--
		}
	}

	indices := make([]int, 0, len(keep))
	for idx := range keep {
		indices = append(indices, idx)
	}
	sortInts(indices)
	return compactSelection{indices: indices}
}

func selectClusterRepresentatives(normed [][]float32, members []int, centroid []float32, quota int) []int {
	if quota >= len(members) {
		out := make([]int, len(members))
		copy(out, members)
		return out
	}

	scored := make([]scoredMember, 0, len(members))
	var sum float64
	for _, idx := range members {
		sim := cosineSim(normed[idx], centroid)
		scored = append(scored, scoredMember{idx: idx, sim: sim})
		sum += float64(sim)
	}
	mean := sum / float64(len(scored))
	var variance float64
	for _, item := range scored {
		diff := float64(item.sim) - mean
		variance += diff * diff
	}
	std := math.Sqrt(variance / float64(len(scored)))
	threshold := float32(mean - std)

	filtered := make([]scoredMember, 0, len(scored))
	for _, item := range scored {
		if item.sim >= threshold {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) < quota {
		filtered = scored
	}

	sortScoredMembers(filtered)

	selected := make([]int, 0, quota)
	seen := make(map[int]struct{}, quota)
	appendIfNeeded := func(idx int) {
		if len(selected) >= quota {
			return
		}
		if _, ok := seen[idx]; ok {
			return
		}
		seen[idx] = struct{}{}
		selected = append(selected, idx)
	}

	appendIfNeeded(filtered[0].idx) // strong core anchor
	recentIdx := filtered[0].idx
	for _, item := range filtered {
		if item.idx > recentIdx {
			recentIdx = item.idx
		}
	}
	appendIfNeeded(recentIdx)

	for len(selected) < quota {
		bestIdx := -1
		bestScore := float32(-1)
		for _, cand := range filtered {
			if _, ok := seen[cand.idx]; ok {
				continue
			}
			minDist := float32(math.MaxFloat32)
			for _, kept := range selected {
				dist := cosineDistance(normed[cand.idx], normed[kept])
				if dist < minDist {
					minDist = dist
				}
			}
			score := minDist + (cand.sim * 0.05)
			if score > bestScore {
				bestScore = score
				bestIdx = cand.idx
			}
		}
		if bestIdx < 0 {
			break
		}
		appendIfNeeded(bestIdx)
	}

	if len(selected) < quota {
		for _, item := range filtered {
			appendIfNeeded(item.idx)
		}
	}

	sortInts(selected)
	return selected
}

func sortClustersForCompaction(clusters []*compactCluster) {
	for i := 1; i < len(clusters); i++ {
		cur := clusters[i]
		j := i - 1
		for ; j >= 0; j-- {
			prev := clusters[j]
			if len(prev.members) > len(cur.members) {
				break
			}
			if len(prev.members) == len(cur.members) && prev.recent > cur.recent {
				break
			}
			clusters[j+1] = clusters[j]
		}
		clusters[j+1] = cur
	}
}

func sortScoredMembers(items []scoredMember) {
	for i := 1; i < len(items); i++ {
		cur := items[i]
		j := i - 1
		for ; j >= 0; j-- {
			prev := items[j]
			if prev.sim > cur.sim {
				break
			}
			if prev.sim == cur.sim && prev.idx < cur.idx {
				break
			}
			items[j+1] = items[j]
		}
		items[j+1] = cur
	}
}

func sortInts(items []int) {
	for i := 1; i < len(items); i++ {
		cur := items[i]
		j := i - 1
		for ; j >= 0 && items[j] > cur; j-- {
			items[j+1] = items[j]
		}
		items[j+1] = cur
	}
}
