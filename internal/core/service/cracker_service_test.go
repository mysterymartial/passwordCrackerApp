package service

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/mocks"
	"testing"
	"time"
)

func TestCrackingService(t *testing.T) {
	tests := []struct {
		name     string
		hash     string
		settings domain.CrackingSettings
		wantErr  bool
		setup    func(*mocks.MockRepository, *mocks.MockHashService)
	}{
		{
			name: "successful_crack_all_algorithms",
			hash: "5f4dcc3b5aa765d61d8327deb882cf99",
			settings: domain.CrackingSettings{
				MinLength:    8,
				MaxLength:    8,
				CharacterSet: domain.CharsetAll,
			},
			setup: func(repo *mocks.MockRepository, hash *mocks.MockHashService) {
				hash.On("Verify", "password", mock.Anything).Return(true)
			},
		},
		{
			name: "concurrent_execution",
			hash: "5f4dcc3b5aa765d61d8327deb882cf99",
			settings: domain.CrackingSettings{
				UseWordlist:   true,
				WordlistPaths: []string{"testdata/wordlist.txt"},
			},
			setup: func(repo *mocks.MockRepository, hash *mocks.MockHashService) {
				hash.On("Verify", mock.Anything, mock.Anything).Return(false)
			},
		},
		{
			name: "early_termination",
			hash: "202cb962ac59075b964b07152d234b70",
			settings: domain.CrackingSettings{
				MinLength: 3,
				MaxLength: 3,
			},
			setup: func(repo *mocks.MockRepository, hash *mocks.MockHashService) {
				hash.On("Verify", "123", mock.Anything).Return(true)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockRepository()
			mockHash := mocks.NewMockHashService()
			tt.setup(mockRepo, mockHash)

			svc := NewCrackingService(mockRepo, mockHash)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			job, err := svc.StartCracking(ctx, tt.hash, tt.settings)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, job)

			// Wait for job completion
			time.Sleep(2 * time.Second)

			progress, err := svc.GetProgress(ctx, job.ID)
			assert.NoError(t, err)
			assert.Equal(t, 6, progress.ActiveThreads)

			completedJob, _ := mockRepo.GetJob(ctx, job.ID)
			assert.NotEqual(t, domain.StatusRunning, completedJob.Status)
		})
	}
}

func TestMetricsCollection(t *testing.T) {
	mockRepo := mocks.NewMockRepository()
	mockHash := mocks.NewMockHashService()
	svc := NewCrackingService(mockRepo, mockHash)

	job, _ := svc.StartCracking(context.Background(), "hash", domain.CrackingSettings{})

	time.Sleep(1 * time.Second)
	progress, _ := svc.GetProgress(context.Background(), job.ID)

	assert.NotZero(t, progress.Speed)
	assert.NotZero(t, progress.Progress)
	assert.Equal(t, 6, progress.ActiveThreads)
}
