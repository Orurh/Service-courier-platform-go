package handlers

import "course-go-avito-Orurh/internal/domain"

func (req createCourierRequest) toModel() *domain.Courier {
	return &domain.Courier{
		Name:          req.Name,
		Phone:         req.Phone,
		Status:        req.Status,
		TransportType: req.TransportType,
	}
}

func (req updateCourierRequest) toModel() domain.PartialCourierUpdate {
	return domain.PartialCourierUpdate{
		ID:            req.ID,
		Name:          req.Name,
		Phone:         req.Phone,
		Status:        req.Status,
		TransportType: req.TransportType,
	}
}

func modelToResponse(c domain.Courier) courierDTO {
	return courierDTO{
		ID:            c.ID,
		Name:          c.Name,
		Phone:         c.Phone,
		Status:        c.Status,
		TransportType: c.TransportType,
	}
}

func modelsToResponse(list []domain.Courier) []courierDTO {
	out := make([]courierDTO, 0, len(list))
	for _, c := range list {
		out = append(out, modelToResponse(c))
	}
	return out
}
