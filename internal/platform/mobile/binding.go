package mobile

import (
	"context"
	"encoding/json"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/core/service"
)

type MobileBinding struct {
	crackingService *service.CrackingService
}

// For iOS/Android
func (m *MobileBinding) StartCracking(hashStr string, settingsJson string) string {
	var settings domain.CrackingSettings
	json.Unmarshal([]byte(settingsJson), &settings)

	job, _ := m.crackingService.StartCracking(context.Background(), hashStr, settings)

	result, _ := json.Marshal(job)
	return string(result)
}

func (m *MobileBinding) GetProgress(jobId string) string {
	progress, _ := m.crackingService.GetProgress(context.Background(), jobId)

	result, _ := json.Marshal(progress)
	return string(result)
}
