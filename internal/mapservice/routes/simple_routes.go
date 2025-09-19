package routes

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"go-falcon/internal/mapservice/dto"
	"go-falcon/internal/mapservice/services"
)

// RegisterStatusRoute registers the module status endpoint
func RegisterStatusRoute(api huma.API, service *services.MapService) {
	huma.Register(api, huma.Operation{
		OperationID: "getMapStatus",
		Method:      http.MethodGet,
		Path:        "/map/status",
		Summary:     "Get map module status",
		Description: "Returns the health status and statistics of the map module",
		Tags:        []string{"Module Status"},
	}, func(ctx context.Context, input *struct{}) (*dto.MapStatusOutput, error) {
		return service.GetModuleStatus(ctx)
	})
}

// RegisterSearchRoute registers the system search endpoint
func RegisterSearchRoute(api huma.API, service *services.MapService) {
	huma.Register(api, huma.Operation{
		OperationID: "searchSystems",
		Method:      http.MethodGet,
		Path:        "/map/search",
		Summary:     "Search for systems",
		Description: "Search for EVE Online systems by name",
		Tags:        []string{"Map"},
	}, func(ctx context.Context, input *dto.SearchSystemInput) (*struct {
		Body []dto.SearchSystemOutput `json:"results"`
	}, error) {
		results, err := service.SearchSystems(ctx, input.Query, input.Limit)
		if err != nil {
			return nil, err
		}
		return &struct {
			Body []dto.SearchSystemOutput `json:"results"`
		}{Body: results}, nil
	})
}

// RegisterRegionRoute registers the region data endpoint
func RegisterRegionRoute(api huma.API, service *services.MapService) {
	huma.Register(api, huma.Operation{
		OperationID: "getRegionData",
		Method:      http.MethodGet,
		Path:        "/map/region/{regionId}",
		Summary:     "Get region map data",
		Description: "Returns all systems and connections for a region as an array of map elements",
		Tags:        []string{"Map"},
	}, func(ctx context.Context, input *struct {
		RegionId int32 `path:"regionId" doc:"EVE Region ID"`
	}) (*struct {
		Body *dto.MapRegionOutput
	}, error) {
		result, err := service.GetRegionSystems(ctx, input.RegionId)
		if err != nil {
			return nil, err
		}
		return &struct {
			Body *dto.MapRegionOutput
		}{Body: result}, nil
	})
}

// RegisterRouteCalculation registers the route calculation endpoint
func RegisterRouteCalculation(api huma.API, service *services.RouteService) {
	huma.Register(api, huma.Operation{
		OperationID: "calculateRoute",
		Method:      http.MethodGet,
		Path:        "/map/route",
		Summary:     "Calculate route",
		Description: "Calculate a route between two systems",
		Tags:        []string{"Map"},
	}, func(ctx context.Context, input *dto.RouteInput) (*dto.RouteOutput, error) {
		return service.CalculateRoute(ctx, *input)
	})
}
