package services

import (
	"container/heap"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go-falcon/internal/mapservice/dto"
	"go-falcon/internal/mapservice/models"
	"go-falcon/pkg/sde"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// RouteService handles route calculation and pathfinding
type RouteService struct {
	db    *mongo.Database
	redis *redis.Client
	sde   *sde.Service
}

// NewRouteService creates a new route service
func NewRouteService(db *mongo.Database, redis *redis.Client, sdeService *sde.Service) *RouteService {
	return &RouteService{
		db:    db,
		redis: redis,
		sde:   sdeService,
	}
}

// RouteNode represents a system in the routing graph
type RouteNode struct {
	SystemID int32
	Distance float64
	Previous *RouteNode
	Index    int // Index in the heap
}

// PriorityQueue implements heap.Interface for Dijkstra's algorithm
type PriorityQueue []*RouteNode

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Distance < pq[j].Distance
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*RouteNode)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// CalculateRoute calculates the optimal route between two systems
func (s *RouteService) CalculateRoute(ctx context.Context, input dto.RouteInput) (*dto.RouteOutput, error) {
	// Check cache first
	cacheKey := s.getRouteCacheKey(input)
	cachedRoute, err := s.getRouteFromCache(ctx, cacheKey)
	if err == nil && cachedRoute != nil {
		return cachedRoute, nil
	}

	// Build the routing graph
	graph, err := s.buildRoutingGraph(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to build routing graph: %w", err)
	}

	// Run Dijkstra's algorithm
	path := s.dijkstra(graph, input.FromSystemID, input.ToSystemID, input)
	if path == nil {
		return nil, fmt.Errorf("no route found between systems")
	}

	// Build the route output
	output, err := s.buildRouteOutput(ctx, input, path)
	if err != nil {
		return nil, fmt.Errorf("failed to build route output: %w", err)
	}

	// Cache the result
	s.cacheRoute(ctx, cacheKey, output)

	return output, nil
}

// buildRoutingGraph creates a graph of system connections
func (s *RouteService) buildRoutingGraph(ctx context.Context, input dto.RouteInput) (map[int32][]int32, error) {
	graph := make(map[int32][]int32)

	// Get all solar systems
	allSystems, err := s.sde.GetAllSolarSystems()
	if err != nil {
		return nil, fmt.Errorf("failed to get solar systems: %w", err)
	}

	// Build gate connections
	for systemID, system := range allSystems {
		// Skip systems we want to avoid
		if s.shouldAvoidSystem(int32(systemID), input.AvoidSystemIDs) {
			continue
		}

		// Add gate connections from stargates
		if system.Stargates != nil {
			for _, stargate := range system.Stargates {
				if stargate.Destination > 0 {
					// Get destination system to find which system the destination gate is in
					destGate, err := s.getStargateByID(stargate.Destination)
					if err != nil || destGate == nil {
						continue
					}

					destSystemID := destGate.SystemID

					// Check if destination should be avoided
					if s.shouldAvoidSystem(destSystemID, input.AvoidSystemIDs) {
						continue
					}

					// Apply route type filters
					if !s.isConnectionAllowed(int32(systemID), destSystemID, input.RouteType) {
						continue
					}

					graph[int32(systemID)] = append(graph[int32(systemID)], destSystemID)
				}
			}
		}
	}

	// Add wormhole connections if requested
	if input.IncludeWH {
		wormholes, err := s.getActiveWormholes(ctx)
		if err == nil {
			for _, wh := range wormholes {
				// Check if systems should be avoided
				if !s.shouldAvoidSystem(wh.FromSystemID, input.AvoidSystemIDs) &&
					!s.shouldAvoidSystem(wh.ToSystemID, input.AvoidSystemIDs) {

					// Add bidirectional connection
					graph[wh.FromSystemID] = append(graph[wh.FromSystemID], wh.ToSystemID)
					graph[wh.ToSystemID] = append(graph[wh.ToSystemID], wh.FromSystemID)
				}
			}
		}
	}

	// Add Thera connections if requested
	if input.IncludeThera {
		// TODO: Integrate Eve-Scout API for Thera connections
		// For now, we'll just skip this
	}

	return graph, nil
}

// dijkstra implements Dijkstra's shortest path algorithm
func (s *RouteService) dijkstra(graph map[int32][]int32, start, end int32, input dto.RouteInput) []int32 {
	// Initialize distance map and priority queue
	distances := make(map[int32]float64)
	visited := make(map[int32]bool)
	previous := make(map[int32]int32)

	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	// Add start node
	startNode := &RouteNode{
		SystemID: start,
		Distance: 0,
	}
	heap.Push(&pq, startNode)
	distances[start] = 0

	for pq.Len() > 0 {
		current := heap.Pop(&pq).(*RouteNode)

		// Check if we reached the destination
		if current.SystemID == end {
			// Reconstruct path
			path := []int32{}
			for systemID := end; systemID != start; {
				path = append([]int32{systemID}, path...)
				prevSystem, exists := previous[systemID]
				if !exists {
					break
				}
				systemID = prevSystem
			}
			path = append([]int32{start}, path...)
			return path
		}

		// Skip if already visited
		if visited[current.SystemID] {
			continue
		}
		visited[current.SystemID] = true

		// Check all neighbors
		for _, neighbor := range graph[current.SystemID] {
			if visited[neighbor] {
				continue
			}

			// Calculate distance (weight) based on route type
			weight := s.calculateEdgeWeight(current.SystemID, neighbor, input.RouteType)
			newDistance := current.Distance + weight

			// Update if we found a shorter path
			if oldDistance, exists := distances[neighbor]; !exists || newDistance < oldDistance {
				distances[neighbor] = newDistance
				previous[neighbor] = current.SystemID

				neighborNode := &RouteNode{
					SystemID: neighbor,
					Distance: newDistance,
				}
				heap.Push(&pq, neighborNode)
			}
		}
	}

	// No path found
	return nil
}

// calculateEdgeWeight calculates the weight of an edge based on route preferences
func (s *RouteService) calculateEdgeWeight(from, to int32, routeType string) float64 {
	// Base weight is 1 jump
	weight := 1.0

	// Get system information
	fromSystem, _ := s.sde.GetSolarSystem(int(from))
	toSystem, _ := s.sde.GetSolarSystem(int(to))

	if fromSystem == nil || toSystem == nil {
		return weight
	}

	switch routeType {
	case "safest":
		// Prefer high-sec systems
		if toSystem.Security < 0.5 {
			weight += 10.0 * (0.5 - toSystem.Security)
		}
		if toSystem.Security <= 0.0 {
			weight += 100.0 // Heavily penalize null-sec
		}

	case "avoid_null":
		// Avoid null-sec systems
		if toSystem.Security <= 0.0 {
			weight += 1000.0 // Effectively avoid null-sec
		}

	case "shortest":
		// Default weight - just count jumps
		// Already set to 1.0
	}

	return weight
}

// isConnectionAllowed checks if a connection is allowed based on route type
func (s *RouteService) isConnectionAllowed(systemID int32, destSystemID int32, routeType string) bool {
	system, err := s.sde.GetSolarSystem(int(systemID))
	if err != nil || system == nil {
		return false
	}

	destSystem, err := s.sde.GetSolarSystem(int(destSystemID))
	if err != nil || destSystem == nil {
		return false
	}

	switch routeType {
	case "safest":
		// Allow all connections, but weight will penalize low-sec
		return true

	case "avoid_null":
		// Don't completely block null-sec, just heavily penalize
		return true

	case "shortest":
		// Allow all connections
		return true
	}

	return true
}

// shouldAvoidSystem checks if a system should be avoided
func (s *RouteService) shouldAvoidSystem(systemID int32, avoidList []int32) bool {
	for _, avoidID := range avoidList {
		if systemID == avoidID {
			return true
		}
	}
	return false
}

// getActiveWormholes retrieves currently active wormhole connections
func (s *RouteService) getActiveWormholes(ctx context.Context) ([]models.MapWormhole, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"expires_at": bson.M{"$gt": time.Now()}},
			{"expires_at": nil},
		},
		"sharing_level": "alliance", // Only use publicly shared wormholes for routing
	}

	cursor, err := s.db.Collection("map_wormholes").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var wormholes []models.MapWormhole
	if err := cursor.All(ctx, &wormholes); err != nil {
		return nil, err
	}

	return wormholes, nil
}

// buildRouteOutput builds the route output DTO
func (s *RouteService) buildRouteOutput(ctx context.Context, input dto.RouteInput, path []int32) (*dto.RouteOutput, error) {
	// Get system information for origin and destination
	fromSystem, _ := s.sde.GetSolarSystem(int(input.FromSystemID))
	toSystem, _ := s.sde.GetSolarSystem(int(input.ToSystemID))

	// Build route systems
	routeSystems := make([]dto.RouteSystem, 0, len(path))
	securityStats := dto.SecurityStats{}

	for _, systemID := range path {
		system, err := s.sde.GetSolarSystem(int(systemID))
		if err != nil {
			continue
		}

		// Get constellation and region info
		constellationID := getConstellationIDForSystem(s.sde, system)
		constellation, _ := s.sde.GetConstellation(int(constellationID))
		var region *sde.Region
		if constellation != nil {
			regionID := getRegionIDForConstellation(s.sde, constellation)
			region, _ = s.sde.GetRegion(int(regionID))
		}

		routeSystem := dto.RouteSystem{
			SystemID:   systemID,
			SystemName: s.getSystemName(system),
			Security:   float32(system.Security),
		}

		if region != nil {
			routeSystem.RegionID = int32(region.RegionID)
			routeSystem.RegionName = GetRegionName(s.sde, region)
		}

		// Check if this is a wormhole connection
		if len(routeSystems) > 0 {
			prevSystemID := routeSystems[len(routeSystems)-1].SystemID
			if s.isWormholeConnection(ctx, prevSystemID, systemID) {
				routeSystem.IsWormhole = true
			}
		}

		routeSystems = append(routeSystems, routeSystem)

		// Update security stats
		if system.Security >= 0.5 {
			securityStats.HighSec++
		} else if system.Security > 0.0 {
			securityStats.LowSec++
		} else if system.Security <= 0.0 {
			if s.isWormholeSystem(system) {
				securityStats.Wormhole++
			} else {
				securityStats.NullSec++
			}
		}
	}

	output := &dto.RouteOutput{
		FromSystemID:      input.FromSystemID,
		FromSystemName:    s.getSystemName(fromSystem),
		ToSystemID:        input.ToSystemID,
		ToSystemName:      s.getSystemName(toSystem),
		RouteType:         input.RouteType,
		Route:             routeSystems,
		Jumps:             len(path) - 1, // Subtract 1 because path includes start system
		IncludesWH:        false,         // Will be set if route uses wormholes
		IncludesThera:     false,         // Will be set if route uses Thera
		SecurityBreakdown: securityStats,
	}

	// Check if route includes wormholes
	for _, system := range routeSystems {
		if system.IsWormhole {
			output.IncludesWH = true
			break
		}
	}

	return output, nil
}

// isWormholeConnection checks if two systems are connected by a wormhole
func (s *RouteService) isWormholeConnection(ctx context.Context, from, to int32) bool {
	filter := bson.M{
		"$and": []bson.M{
			{
				"$or": []bson.M{
					{
						"from_system_id": from,
						"to_system_id":   to,
					},
					{
						"from_system_id": to,
						"to_system_id":   from,
					},
				},
			},
			{
				"$or": []bson.M{
					{"expires_at": bson.M{"$gt": time.Now()}},
					{"expires_at": nil},
				},
			},
		},
	}

	count, err := s.db.Collection("map_wormholes").CountDocuments(ctx, filter)
	return err == nil && count > 0
}

// isWormholeSystem checks if a system is a wormhole system
func (s *RouteService) isWormholeSystem(system *sde.SolarSystem) bool {
	// Wormhole systems typically have IDs in specific ranges
	// J-space systems: 31000000 - 32000000
	// Thera: 31000005
	// Shattered wormholes have specific patterns
	return system.SolarSystemID >= 31000000 && system.SolarSystemID < 32000000
}

// getStargateByID finds which system a stargate belongs to
type StargateInfo struct {
	SystemID int32
	Stargate *sde.Stargate
}

func (s *RouteService) getStargateByID(stargateID int) (*StargateInfo, error) {
	// We need to search through all systems to find which one contains this stargate
	allSystems, err := s.sde.GetAllSolarSystems()
	if err != nil {
		return nil, err
	}

	stargateIDStr := fmt.Sprintf("%d", stargateID)
	for systemID, system := range allSystems {
		if system.Stargates != nil {
			if stargate, exists := system.Stargates[stargateIDStr]; exists {
				return &StargateInfo{
					SystemID: int32(systemID),
					Stargate: stargate,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("stargate %d not found", stargateID)
}

// Cache management

func (s *RouteService) getRouteCacheKey(input dto.RouteInput) string {
	// Create a unique cache key based on input parameters
	return fmt.Sprintf("route:%d:%d:%s:%t:%t:%v",
		input.FromSystemID,
		input.ToSystemID,
		input.RouteType,
		input.IncludeWH,
		input.IncludeThera,
		input.AvoidSystemIDs,
	)
}

func (s *RouteService) getRouteFromCache(ctx context.Context, key string) (*dto.RouteOutput, error) {
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var route dto.RouteOutput
	if err := json.Unmarshal([]byte(data), &route); err != nil {
		return nil, err
	}

	return &route, nil
}

func (s *RouteService) cacheRoute(ctx context.Context, key string, route *dto.RouteOutput) {
	data, err := json.Marshal(route)
	if err != nil {
		return
	}

	// Cache for 5 minutes
	s.redis.Set(ctx, key, data, 5*time.Minute)
}

// Helper methods for getting system information

func (s *RouteService) getSystemName(system *sde.SolarSystem) string {
	return GetSystemName(s.sde, system)
}

// SaveRoute saves a calculated route to the database
func (s *RouteService) SaveRoute(ctx context.Context, route *dto.RouteOutput) error {
	// Convert route to database model
	systemIDs := make([]int32, len(route.Route))
	for i, system := range route.Route {
		systemIDs[i] = system.SystemID
	}

	dbRoute := &models.MapRoute{
		FromSystemID:  route.FromSystemID,
		ToSystemID:    route.ToSystemID,
		RouteType:     route.RouteType,
		Route:         systemIDs,
		Jumps:         route.Jumps,
		IncludesWH:    route.IncludesWH,
		IncludesThera: route.IncludesThera,
		CachedAt:      time.Now(),
		ExpiresAt:     time.Now().Add(1 * time.Hour),
	}

	_, err := s.db.Collection("map_routes").InsertOne(ctx, dbRoute)
	return err
}
