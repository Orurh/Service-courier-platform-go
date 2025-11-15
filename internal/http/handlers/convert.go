package handlers

import "course-go-avito-Orurh/internal/domain"

func (r createCourierRequest) toModel() *domain.Courier {
	return &domain.Courier{
		Name:   r.Name,
		Phone:  r.Phone,
		Status: r.Status,
	}
}

func (r updateCourierRequest) toModel() domain.PartialCourierUpdate {
	return domain.PartialCourierUpdate{
		ID:     r.ID,
		Name:   r.Name,
		Phone:  r.Phone,
		Status: r.Status,
	}
}

func modelToResponse(c domain.Courier) courierDTO {
	return courierDTO{
		ID:     c.ID,
		Name:   c.Name,
		Phone:  c.Phone,
		Status: c.Status,
	}
}

func modelsToResponse(list []domain.Courier) []courierDTO {
	out := make([]courierDTO, 0, len(list))
	for _, c := range list {
		out = append(out, modelToResponse(c))
	}
	return out
}
