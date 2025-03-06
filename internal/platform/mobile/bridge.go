package mobile

import (
	"encoding/json"
)

// Bridge structs for mobile data transfer
type MobileResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func createErrorResponse(err error) string {
	response := MobileResponse{
		Success: false,
		Error:   err.Error(),
	}
	result, _ := json.Marshal(response)
	return string(result)
}

func createSuccessResponse(data interface{}) string {
	response := MobileResponse{
		Success: true,
		Data:    data,
	}
	result, _ := json.Marshal(response)
	return string(result)
}
