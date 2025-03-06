package desktop

import (
	"context"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/core/service"
)

type DesktopLib struct {
	crackingService *service.CrackingService
	config          *Config
}

func NewDesktopLib(svc *service.CrackingService, cfg *Config) *DesktopLib {
	return &DesktopLib{
		crackingService: svc,
		config:          cfg,
	}
}

// Direct library methods for desktop applications
func (d *DesktopLib) StartCracking(hash string, settings domain.CrackingSettings) (*domain.CrackingJob, error) {
	ctx := context.Background()
	return d.crackingService.StartCracking(ctx, hash, settings)
}

func (d *DesktopLib) GetProgress(jobID string) (*domain.JobProgress, error) {
	ctx := context.Background()
	return d.crackingService.GetProgress(ctx, jobID)
}

func (d *DesktopLib) StopCracking(jobID string) error {
	ctx := context.Background()
	return d.crackingService.StopCracking(ctx, jobID)
}

func (d *DesktopLib) GetResults(jobID string) (*domain.CrackingResult, error) {
	ctx := context.Background()
	return d.crackingService.GetResults(ctx, jobID)
}
