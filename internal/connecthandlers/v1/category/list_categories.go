package v1Category

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	category "github.com/carson-networks/budget-server/internal/connecthandlers/gen/category/v1"
	"github.com/carson-networks/budget-server/internal/logging"
	storagecategory "github.com/carson-networks/budget-server/internal/storage/category"
)

// ListCategories implements category.v1.CategoryService.ListCategories.
func (s *Service) ListCategories(ctx context.Context, req *connect.Request[category.ListCategoriesRequest]) (*connect.Response[category.ListCategoriesResponse], error) {
	logData := logging.GetLogData(ctx)
	limit := 20
	offset := 0
	if c := req.Msg.GetCursor(); c != nil {
		if c.GetPosition() < 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cursor position must be non-negative"))
		}
		offset = int(c.GetPosition())
		if c.GetLimit() > 0 {
			limit = int(c.GetLimit())
		}
	}
	filter := &storagecategory.CategoryFilter{
		Limit:  limit,
		Offset: offset,
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("listCategoriesMs")
	}
	result, err := s.Storage.Read().Categories.List(ctx, filter)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	categories := result.Categories
	if categories == nil {
		categories = []*storagecategory.Category{}
	}
	if logData != nil {
		logData.AddData("categoryCount", len(categories))
	}

	out := &category.ListCategoriesResponse{
		Categories: make([]*category.Category, len(categories)),
	}
	for i, cat := range categories {
		apiCat := &category.Category{
			Id:           cat.ID.String(),
			Name:         cat.Name,
			IsParent:     cat.IsParent,
			IsDisabled:   cat.IsDisabled,
			CategoryType: toConnectCategoryType(cat.CategoryType),
			CreatedAt:    timestamppb.New(cat.CreatedAt),
		}
		if cat.ParentCategoryID != nil {
			pid := cat.ParentCategoryID.String()
			apiCat.ParentCategoryId = &pid
		}
		out.Categories[i] = apiCat
	}
	if result.NextCursor != nil {
		out.NextCursor = &category.ListCategoriesCursor{
			Position: int32(result.NextCursor.Position),
			Limit:    int32(result.NextCursor.Limit),
		}
	}
	return connect.NewResponse(out), nil
}
