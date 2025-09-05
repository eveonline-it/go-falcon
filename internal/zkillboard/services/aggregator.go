package services

import (
	"context"
	"fmt"
	"time"

	killmailModels "go-falcon/internal/killmails/models"
	"go-falcon/internal/zkillboard/models"
	"go-falcon/pkg/sde"
	"go.mongodb.org/mongo-driver/bson"
)

// Aggregator handles timeseries data aggregation for killmails
type Aggregator struct {
	repository *Repository
	sdeService sde.SDEService
}

// NewAggregator creates a new aggregator instance
func NewAggregator(repository *Repository, sdeService sde.SDEService) *Aggregator {
	return &Aggregator{
		repository: repository,
		sdeService: sdeService,
	}
}

// UpdateTimeseries updates all relevant timeseries aggregations for a killmail
func (a *Aggregator) UpdateTimeseries(ctx context.Context, km *killmailModels.Killmail, zkb *models.ZKBMetadata) error {
	// Get location information from SDE
	regionID, constellationID := a.getLocationInfo(km.SolarSystemID)

	// Update hourly aggregation
	if err := a.updatePeriodAggregation(ctx, "hour", km, zkb, regionID, constellationID); err != nil {
		return fmt.Errorf("failed to update hourly aggregation: %w", err)
	}

	// Update daily aggregation
	if err := a.updatePeriodAggregation(ctx, "day", km, zkb, regionID, constellationID); err != nil {
		return fmt.Errorf("failed to update daily aggregation: %w", err)
	}

	// Update monthly aggregation
	if err := a.updatePeriodAggregation(ctx, "month", km, zkb, regionID, constellationID); err != nil {
		return fmt.Errorf("failed to update monthly aggregation: %w", err)
	}

	return nil
}

// updatePeriodAggregation updates a specific period's aggregation
func (a *Aggregator) updatePeriodAggregation(
	ctx context.Context,
	period string,
	km *killmailModels.Killmail,
	zkb *models.ZKBMetadata,
	regionID, constellationID int32,
) error {
	// Truncate timestamp based on period
	timestamp := a.truncateTimestamp(km.KillmailTime, period)

	// System-level aggregation
	if err := a.updateSystemAggregation(ctx, period, timestamp, km, zkb, regionID, constellationID); err != nil {
		return err
	}

	// Region-level aggregation
	if regionID != 0 {
		if err := a.updateRegionAggregation(ctx, period, timestamp, km, zkb, regionID); err != nil {
			return err
		}
	}

	// Alliance-level aggregation (for victim)
	if km.Victim.AllianceID != nil && *km.Victim.AllianceID != 0 {
		if err := a.updateAllianceAggregation(ctx, period, timestamp, km, zkb, int32(*km.Victim.AllianceID), true); err != nil {
			return err
		}
	}

	// Alliance-level aggregation (for attackers)
	for _, attacker := range km.Attackers {
		if attacker.AllianceID != nil && *attacker.AllianceID != 0 {
			if err := a.updateAllianceAggregation(ctx, period, timestamp, km, zkb, int32(*attacker.AllianceID), false); err != nil {
				return err
			}
		}
	}

	// Corporation-level aggregation (for victim)
	if km.Victim.CorporationID != nil {
		if err := a.updateCorporationAggregation(ctx, period, timestamp, km, zkb, int32(*km.Victim.CorporationID), true); err != nil {
			return err
		}
	}

	// Corporation-level aggregation (for attackers)
	for _, attacker := range km.Attackers {
		if attacker.CorporationID != nil && *attacker.CorporationID != 0 {
			if err := a.updateCorporationAggregation(ctx, period, timestamp, km, zkb, int32(*attacker.CorporationID), false); err != nil {
				return err
			}
		}
	}

	// Ship type aggregation
	if err := a.updateShipTypeAggregation(ctx, period, timestamp, km, zkb); err != nil {
		return err
	}

	return nil
}

// updateSystemAggregation updates system-level timeseries
func (a *Aggregator) updateSystemAggregation(
	ctx context.Context,
	period string,
	timestamp time.Time,
	km *killmailModels.Killmail,
	zkb *models.ZKBMetadata,
	regionID, constellationID int32,
) error {
	filter := bson.M{
		"period":          period,
		"timestamp":       timestamp,
		"solar_system_id": km.SolarSystemID,
	}

	increments := bson.M{
		"kill_count":  1,
		"total_value": zkb.TotalValue,
	}

	if zkb.NPC {
		increments["npc_kills"] = 1
	}
	if zkb.Solo {
		increments["solo_kills"] = 1
	}

	// Increment ship type counter
	shipTypeKey := fmt.Sprintf("ship_types.%d", km.Victim.ShipTypeID)
	increments[shipTypeKey] = 1

	// Set location IDs (currently unused but prepared for future use)
	_ = constellationID // Will be used when location mapping is implemented
	_ = regionID        // Will be used when location mapping is implemented

	return a.repository.IncrementTimeseries(ctx, filter, increments)
}

// updateRegionAggregation updates region-level timeseries
func (a *Aggregator) updateRegionAggregation(
	ctx context.Context,
	period string,
	timestamp time.Time,
	km *killmailModels.Killmail,
	zkb *models.ZKBMetadata,
	regionID int32,
) error {
	filter := bson.M{
		"period":    period,
		"timestamp": timestamp,
		"region_id": regionID,
	}

	increments := bson.M{
		"kill_count":  1,
		"total_value": zkb.TotalValue,
	}

	if zkb.NPC {
		increments["npc_kills"] = 1
	}
	if zkb.Solo {
		increments["solo_kills"] = 1
	}

	return a.repository.IncrementTimeseries(ctx, filter, increments)
}

// updateAllianceAggregation updates alliance-level timeseries
func (a *Aggregator) updateAllianceAggregation(
	ctx context.Context,
	period string,
	timestamp time.Time,
	km *killmailModels.Killmail,
	zkb *models.ZKBMetadata,
	allianceID int32,
	isVictim bool,
) error {
	filter := bson.M{
		"period":      period,
		"timestamp":   timestamp,
		"alliance_id": allianceID,
	}

	increments := bson.M{
		"total_value": zkb.TotalValue,
	}

	if isVictim {
		increments["losses"] = 1
	} else {
		increments["kill_count"] = 1
	}

	return a.repository.IncrementTimeseries(ctx, filter, increments)
}

// updateCorporationAggregation updates corporation-level timeseries
func (a *Aggregator) updateCorporationAggregation(
	ctx context.Context,
	period string,
	timestamp time.Time,
	km *killmailModels.Killmail,
	zkb *models.ZKBMetadata,
	corporationID int32,
	isVictim bool,
) error {
	filter := bson.M{
		"period":         period,
		"timestamp":      timestamp,
		"corporation_id": corporationID,
	}

	increments := bson.M{
		"total_value": zkb.TotalValue,
	}

	if isVictim {
		increments["losses"] = 1
	} else {
		increments["kill_count"] = 1
	}

	return a.repository.IncrementTimeseries(ctx, filter, increments)
}

// updateShipTypeAggregation updates ship type timeseries
func (a *Aggregator) updateShipTypeAggregation(
	ctx context.Context,
	period string,
	timestamp time.Time,
	km *killmailModels.Killmail,
	zkb *models.ZKBMetadata,
) error {
	filter := bson.M{
		"period":       period,
		"timestamp":    timestamp,
		"ship_type_id": km.Victim.ShipTypeID,
	}

	increments := bson.M{
		"kill_count":  1,
		"total_value": zkb.TotalValue,
	}

	return a.repository.IncrementTimeseries(ctx, filter, increments)
}

// getLocationInfo returns region and constellation IDs for a solar system
// Note: SDE SolarSystem doesn't include RegionID/ConstellationID directly
// This is a simplified implementation - in production you'd need to traverse the hierarchy
func (a *Aggregator) getLocationInfo(solarSystemID int64) (regionID, constellationID int32) {
	// TODO: Implement proper region/constellation lookup
	// For now, return 0 as these relationships need to be resolved through
	// constellation -> region mapping
	_ = solarSystemID // Acknowledge parameter
	return 0, 0
}

// truncateTimestamp truncates a timestamp to the specified period
func (a *Aggregator) truncateTimestamp(t time.Time, period string) time.Time {
	switch period {
	case "hour":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case "day":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	default:
		return t
	}
}

// GetTopSystems returns the most active systems for a period
func (a *Aggregator) GetTopSystems(ctx context.Context, period string, start, end time.Time, limit int) ([]bson.M, error) {
	filter := bson.M{
		"solar_system_id": bson.M{"$exists": true, "$ne": 0},
	}

	timeseries, err := a.repository.GetTimeseries(ctx, period, start, end, filter)
	if err != nil {
		return nil, err
	}

	// Aggregate by system
	systemStats := make(map[int32]*bson.M)
	for _, ts := range timeseries {
		if ts.SolarSystemID == 0 {
			continue
		}

		if _, exists := systemStats[ts.SolarSystemID]; !exists {
			systemName := ""
			if system, err := a.sdeService.GetSolarSystem(int(ts.SolarSystemID)); err == nil {
				// SolarSystem doesn't have a direct name field
				// TODO: Resolve system name through NameID lookup
				systemName = fmt.Sprintf("System %d", ts.SolarSystemID)
				_ = system // Acknowledge we got the system but can't use name yet
			}

			systemStats[ts.SolarSystemID] = &bson.M{
				"system_id":   ts.SolarSystemID,
				"system_name": systemName,
				"kills":       0,
				"value":       float64(0),
			}
		}

		stats := systemStats[ts.SolarSystemID]
		(*stats)["kills"] = (*stats)["kills"].(int) + ts.KillCount
		(*stats)["value"] = (*stats)["value"].(float64) + ts.TotalValue
	}

	// Convert to slice and sort
	results := make([]bson.M, 0, len(systemStats))
	for _, stats := range systemStats {
		results = append(results, *stats)
	}

	// Sort by kill count (would normally use sort package)
	// Limiting results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetTopAlliances returns the most active alliances for a period
func (a *Aggregator) GetTopAlliances(ctx context.Context, period string, start, end time.Time, limit int) ([]bson.M, error) {
	filter := bson.M{
		"alliance_id": bson.M{"$exists": true, "$ne": 0},
	}

	timeseries, err := a.repository.GetTimeseries(ctx, period, start, end, filter)
	if err != nil {
		return nil, err
	}

	// Aggregate by alliance
	allianceStats := make(map[int32]*bson.M)
	for _, ts := range timeseries {
		if ts.AllianceID == 0 {
			continue
		}

		if _, exists := allianceStats[ts.AllianceID]; !exists {
			allianceStats[ts.AllianceID] = &bson.M{
				"alliance_id":   ts.AllianceID,
				"alliance_name": fmt.Sprintf("Alliance %d", ts.AllianceID), // TODO: Resolve name
				"kills":         0,
				"losses":        0,
				"value":         float64(0),
			}
		}

		stats := allianceStats[ts.AllianceID]
		(*stats)["kills"] = (*stats)["kills"].(int) + ts.KillCount
		(*stats)["value"] = (*stats)["value"].(float64) + ts.TotalValue
	}

	// Convert to slice
	results := make([]bson.M, 0, len(allianceStats))
	for _, stats := range allianceStats {
		results = append(results, *stats)
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetTopShipTypes returns the most destroyed ship types for a period
func (a *Aggregator) GetTopShipTypes(ctx context.Context, period string, start, end time.Time, limit int) ([]bson.M, error) {
	filter := bson.M{
		"ship_type_id": bson.M{"$exists": true, "$ne": 0},
	}

	timeseries, err := a.repository.GetTimeseries(ctx, period, start, end, filter)
	if err != nil {
		return nil, err
	}

	// Aggregate by ship type
	shipStats := make(map[int32]*bson.M)
	for _, ts := range timeseries {
		if ts.ShipTypeID == 0 {
			continue
		}

		if _, exists := shipStats[ts.ShipTypeID]; !exists {
			shipName := ""
			if shipType, err := a.sdeService.GetType(fmt.Sprintf("%d", ts.ShipTypeID)); err == nil {
				if enName, ok := shipType.Name["en"]; ok {
					shipName = enName
				}
			}

			shipStats[ts.ShipTypeID] = &bson.M{
				"ship_type_id":   ts.ShipTypeID,
				"ship_type_name": shipName,
				"destroyed":      0,
				"value":          float64(0),
			}
		}

		stats := shipStats[ts.ShipTypeID]
		(*stats)["destroyed"] = (*stats)["destroyed"].(int) + ts.KillCount
		(*stats)["value"] = (*stats)["value"].(float64) + ts.TotalValue
	}

	// Convert to slice
	results := make([]bson.M, 0, len(shipStats))
	for _, stats := range shipStats {
		results = append(results, *stats)
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}
