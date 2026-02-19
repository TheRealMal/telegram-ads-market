package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"time"

	"github.com/gotd/td/tg"
	"golang.org/x/sync/errgroup"
)

// asyncGraphJob is a field index + token for loading via StatsLoadAsyncGraph.
type asyncGraphJob struct {
	fieldIndex int
	token      string
}

// collectAsyncGraphJobs uses reflection to find StatsGraphClass fields that are
// *tg.StatsGraphAsync and returns jobs to load them.
func collectAsyncGraphJobs(stats *tg.StatsBroadcastStats) []asyncGraphJob {
	statsVal := reflect.ValueOf(stats).Elem()
	graphIface := reflect.TypeOf((*tg.StatsGraphClass)(nil)).Elem()

	var jobs []asyncGraphJob
	for i := 0; i < statsVal.NumField(); i++ {
		f := statsVal.Field(i)
		if f.Type() != graphIface {
			continue
		}
		if f.IsNil() {
			continue
		}
		graph, ok := f.Interface().(tg.StatsGraphClass)
		if !ok {
			continue
		}
		async, ok := graph.(*tg.StatsGraphAsync)
		if !ok {
			continue
		}
		jobs = append(jobs, asyncGraphJob{fieldIndex: i, token: async.Token})
	}
	return jobs
}

func applyLoadedGraphs(stats *tg.StatsBroadcastStats, jobs []asyncGraphJob, results []tg.StatsGraphClass) {
	statsVal := reflect.ValueOf(stats).Elem()
	for i, job := range jobs {
		if results[i] != nil {
			statsVal.Field(job.fieldIndex).Set(reflect.ValueOf(results[i]))
		}
	}
}

func (s *service) UpdateChannelStats(ctx context.Context, channelID int64, accessHash int64) error {
	stats, err := s.telegramClient.API().StatsGetBroadcastStats(ctx, &tg.StatsGetBroadcastStatsRequest{
		Channel: &tg.InputChannel{
			ChannelID:  channelID,
			AccessHash: accessHash,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get broadcast stats: %w", err)
	}

	jobs := collectAsyncGraphJobs(stats)
	slog.Info("collected async graph jobs", "channel_id", channelID, "jobs", len(jobs))
	if len(jobs) > 0 {
		results := make([]tg.StatsGraphClass, len(jobs))
		g, gctx := errgroup.WithContext(ctx)
		api := s.telegramClient.API()
		for i := range jobs {
			g.Go(func() error {
				loaded, err := api.StatsLoadAsyncGraph(gctx, &tg.StatsLoadAsyncGraphRequest{
					Token: jobs[i].token,
				})
				if err != nil {
					return fmt.Errorf("load async graph: %w", err)
				}
				results[i] = loaded
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return fmt.Errorf("loading async graphs: %w", err)
		}
		applyLoadedGraphs(stats, jobs, results)
	}
	slog.Info("applied loaded graphs", "channel_id", channelID)

	jsonStats, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	// Append requested_at timestamp to stats JSON.
	var statsMap map[string]interface{}
	if err := json.Unmarshal(jsonStats, &statsMap); err != nil {
		return fmt.Errorf("failed to unmarshal stats for requested_at: %w", err)
	}
	statsMap["requested_at"] = time.Now().Unix()
	jsonStats, err = json.Marshal(statsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal stats with requested_at: %w", err)
	}

	if err := s.marketRepository.UpsertChannelStats(ctx, channelID, jsonStats); err != nil {
		return fmt.Errorf("failed to upsert channel stats: %w", err)
	}

	return nil
}
