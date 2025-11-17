package handlers

type courierDTO struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
}

type createCourierRequest struct {
	Name   string `json:"name"`
	Phone  string `json:"phone"`
	Status string `json:"status"`
}

type updateCourierRequest struct {
	ID     int64   `json:"id"`
	Name   *string `json:"name,omitempty"`
	Phone  *string `json:"phone,omitempty"`
	Status *string `json:"status,omitempty"`
}
