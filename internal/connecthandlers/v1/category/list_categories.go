package category

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	budgetv1 "github.com/carson-networks/budget-server/gen/budget/v1"
	"github.com/carson-networks/budget-server/internal/logging"
	storagecategory "github.com/carson-networks/budget-server/internal/storage/category"
)

// ListCategories implements budget.v1.CategoryService.ListCategories.
func (s *Service) ListCategories(ctx context.Context, req *connect.Request[budgetv1.ListCategoriesRequest]) (*connect.Response[budgetv1.ListCategoriesResponse], error) {
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

	out := &budgetv1.ListCategoriesResponse{
		Categories: make([]*budgetv1.Category, len(categories)),
	}
	for i, cat := range categories {
		apiCat := &budgetv1.Category{
			Id:           cat.ID.String(),
			Name:         cat.Name,
			IsParent:     cat.IsParent,
			IsDisabled:   cat.IsDisabled,
			CategoryType: budgetv1.CategoryType(cat.CategoryType),
			CreatedAt:    timestamppb.New(cat.CreatedAt),
		}
		if cat.ParentCategoryID != nil {
			pid := cat.ParentCategoryID.String()
			apiCat.ParentCategoryId = &pid
		}
		out.Categories[i] = apiCat
	}
	if result.NextCursor != nil {
		out.NextCursor = &budgetv1.ListCategoriesCursor{
			Position: int32(result.NextCursor.Position),
			Limit:    int32(result.NextCursor.Limit),
		}
	}
	return connect.NewResponse(out), nil
}
