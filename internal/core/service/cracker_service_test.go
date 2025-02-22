package services

//
//import (
//	"context"
//	"errors"
//	"github.com/stretchr/testify/assert"
//	"github.com/stretchr/testify/mock"
//	"passwordCrakerBackend/internal/core/domain"
//	"passwordCrakerBackend/internal/port"
//	"testing"
//	"time"
//)
//
//type MockRepository struct {
//	mock.Mock
//}
//
//func (m *MockRepository) SaveJob(ctx context.Context, job *domain.CrackingJob) error {
//	args := m.Called(ctx, job)
//	return args.Error(0)
//}
//
//func (m *MockRepository) UpdateJob(ctx context.Context, job *domain.CrackingJob) error {
//	args := m.Called(ctx, job)
//	return args.Error(0)
//}
//
//func (m *MockRepository) GetJob(ctx context.Context, jobID string) (*domain.CrackingJob, error) {
//	args := m.Called(ctx, jobID)
//	if args.Get(0) == nil {
//		return nil, args.Error(1)
//	}
//	return args.Get(0).(*domain.CrackingJob), args.Error(1)
//}
//
//func (m *MockRepository) ListJobs(ctx context.Context, filter port.JobFilter) ([]domain.CrackingJob, error) {
//	args := m.Called(ctx, filter)
//	return args.Get(0).([]domain.CrackingJob), args.Error(1)
//}
//
//func (m *MockRepository) SaveMetrics(ctx context.Context, jobID string, metrics *domain.ResourceMetrics) error {
//	args := m.Called(ctx, jobID, metrics)
//	return args.Error(0)
//}
//
//type MockHashService struct {
//	mock.Mock
//}
//
//func (m *MockHashService) Identify(hash string) domain.HashType {
//	args := m.Called(hash)
//	return args.Get(0).(domain.HashType)
//}
//
//func (m *MockHashService) verify(password, hash string, hashType domain.HashType) bool {
//	args := m.Called(password, hash, hashType)
//	return args.Bool(0)
//}
//
//type MockAlgorithm struct {
//	mock.Mock
//	passwordChan chan string
//}
//
//func NewMockAlgorithm() *MockAlgorithm {
//	return &MockAlgorithm{
//		passwordChan: make(chan string),
//	}
//}
//
//func (m *MockAlgorithm) Crack(ctx context.Context, hash string, settings domain.CrackingSettings) <-chan string {
//	m.Called(ctx, hash, settings)
//	return m.passwordChan
//}
//
//func (m *MockAlgorithm) Stop() {
//	m.Called()
//	close(m.passwordChan)
//}
//
//func (m *MockAlgorithm) Name() domain.CrackingAlgorithm {
//	args := m.Called()
//	return args.Get(0).(domain.CrackingAlgorithm)
//}
//
//func TestStartCracking(t *testing.T) {
//	testCases := []struct {
//		name     string
//		hash     string
//		settings domain.CrackingSettings
//		setup    func(*MockRepository, *MockHashService, *MockAlgorithm)
//		wantErr  bool
//		wantJob  bool
//	}{
//		{
//			name: "Successful start",
//			hash: "5f4dcc3b5aa765d61d8327deb882cf99",
//			settings: domain.CrackingSettings{
//				Algorithm: domain.AlgoBruteForce,
//				MinLength: 6,
//				MaxLength: 8,
//				Threads:   4,
//			},
//			setup: func(repo *MockRepository, hash *MockHashService, alg *MockAlgorithm) {
//				hash.On("Identify", mock.Anything).Return(domain.HashMD5)
//				repo.On("SaveJob", mock.Anything, mock.Anything).Return(nil)
//				alg.On("Name").Return(domain.AlgoBruteForce)
//				alg.On("Crack", mock.Anything, mock.Anything, mock.Anything).Return()
//			},
//			wantErr: false,
//			wantJob: true,
//		},
//		{
//			name: "Invalid hash",
//			hash: "invalid",
//			settings: domain.CrackingSettings{
//				Algorithm: domain.AlgoBruteForce,
//			},
//			setup: func(repo *MockRepository, hash *MockHashService, alg *MockAlgorithm) {
//				hash.On("Identify", mock.Anything).Return(domain.HashType(""))
//			},
//			wantErr: true,
//			wantJob: false,
//		},
//		{
//			name: "Repository error",
//			hash: "5f4dcc3b5aa765d61d8327deb882cf99",
//			settings: domain.CrackingSettings{
//				Algorithm: domain.AlgoBruteForce,
//			},
//			setup: func(repo *MockRepository, hash *MockHashService, alg *MockAlgorithm) {
//				hash.On("Identify", mock.Anything).Return(domain.HashMD5)
//				repo.On("SaveJob", mock.Anything, mock.Anything).Return(errors.New("db error"))
//			},
//			wantErr: true,
//			wantJob: false,
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			mockRepo := new(MockRepository)
//			mockHash := new(MockHashService)
//			mockAlg := NewMockAlgorithm()
//
//			tc.setup(mockRepo, mockHash, mockAlg)
//
//			service := NewCrackingService(mockRepo, mockHash, []CrackingAlgorithm{mockAlg})
//
//			job, err := service.StartCracking(context.Background(), tc.hash, tc.settings)
//
//			if tc.wantErr {
//				assert.Error(t, err)
//				assert.Nil(t, job)
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, job)
//				assert.Equal(t, domain.StatusRunning, job.Status)
//				assert.Equal(t, tc.hash, job.TargetHash)
//				assert.NotEmpty(t, job.ID)
//				assert.NotZero(t, job.StartTime)
//			}
//		})
//	}
//}
//
//func TestStopCracking(t *testing.T) {
//	testCases := []struct {
//		name    string
//		jobID   string
//		setup   func(*crackingService, *MockRepository, *MockAlgorithm)
//		wantErr bool
//	}{
//		{
//			name:  "Successfully stop job",
//			jobID: "test-job",
//			setup: func(s *crackingService, repo *MockRepository, alg *MockAlgorithm) {
//				job := &domain.CrackingJob{
//					ID:        "test-job",
//					Algorithm: domain.AlgoBruteForce,
//				}
//				s.activeJobs.Store(job.ID, job)
//				alg.On("Name").Return(domain.AlgoBruteForce)
//				alg.On("Stop").Return()
//				repo.On("UpdateJob", mock.Anything, mock.Anything).Return(nil)
//			},
//			wantErr: false,
//		},
//		{
//			name:    "Job not found",
//			jobID:   "nonexistent",
//			setup:   func(s *crackingService, repo *MockRepository, alg *MockAlgorithm) {},
//			wantErr: true,
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			mockRepo := new(MockRepository)
//			mockHash := new(MockHashService)
//			mockAlg := NewMockAlgorithm()
//
//			service := NewCrackingService(mockRepo, mockHash, []CrackingAlgorithm{mockAlg})
//			tc.setup(service.(*crackingService), mockRepo, mockAlg)
//
//			err := service.StopCracking(context.Background(), tc.jobID)
//
//			if tc.wantErr {
//				assert.Error(t, err)
//			} else {
//				assert.NoError(t, err)
//				val, exists := service.(*crackingService).activeJobs.Load(tc.jobID)
//				assert.True(t, exists)
//				assert.Equal(t, domain.StatusCancelled, val.(*domain.CrackingJob).Status)
//			}
//		})
//	}
//}
//
//func TestGetJobStatus(t *testing.T) {
//	testCases := []struct {
//		name       string
//		jobID      string
//		mockJob    *domain.CrackingJob
//		mockErr    error
//		wantErr    bool
//		wantStatus domain.JobStatus
//	}{
//		{
//			name:  "Get running job status",
//			jobID: "test-job",
//			mockJob: &domain.CrackingJob{
//				ID:     "test-job",
//				Status: domain.StatusRunning,
//			},
//			mockErr:    nil,
//			wantErr:    false,
//			wantStatus: domain.StatusRunning,
//		},
//		{
//			name:       "Job not found",
//			jobID:      "nonexistent",
//			mockJob:    nil,
//			mockErr:    errors.New("not found"),
//			wantErr:    true,
//			wantStatus: "",
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			mockRepo := new(MockRepository)
//			mockHash := new(MockHashService)
//
//			service := NewCrackingService(mockRepo, mockHash, nil)
//
//			mockRepo.On("GetJob", mock.Anything, tc.jobID).Return(tc.mockJob, tc.mockErr)
//
//			status, err := service.GetJobStatus(context.Background(), tc.jobID)
//
//			if tc.wantErr {
//				assert.Error(t, err)
//				assert.Nil(t, status)
//			} else {
//				assert.NoError(t, err)
//				assert.Equal(t, tc.wantStatus, *status)
//			}
//		})
//	}
//}
//
//func TestListJobs(t *testing.T) {
//	testCases := []struct {
//		name      string
//		filter    port.JobFilter
//		mockJobs  []domain.CrackingJob
//		mockErr   error
//		wantErr   bool
//		wantCount int
//	}{
//		{
//			name: "List all jobs",
//			filter: port.JobFilter{
//				Status: domain.StatusRunning,
//				Limit:  10,
//			},
//			mockJobs: []domain.CrackingJob{
//				{ID: "job1", Status: domain.StatusRunning},
//				{ID: "job2", Status: domain.StatusRunning},
//			},
//			mockErr:   nil,
//			wantErr:   false,
//			wantCount: 2,
//		},
//		{
//			name: "Empty result",
//			filter: port.JobFilter{
//				Status: domain.StatusComplete,
//			},
//			mockJobs:  []domain.CrackingJob{},
//			mockErr:   nil,
//			wantErr:   false,
//			wantCount: 0,
//		},
//		{
//			name:      "Repository error",
//			filter:    port.JobFilter{},
//			mockJobs:  nil,
//			mockErr:   errors.New("db error"),
//			wantErr:   true,
//			wantCount: 0,
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			mockRepo := new(MockRepository)
//			mockHash := new(MockHashService)
//
//			service := NewCrackingService(mockRepo, mockHash, nil)
//
//			mockRepo.On("ListJobs", mock.Anything, tc.filter).Return(tc.mockJobs, tc.mockErr)
//
//			jobs, err := service.ListJobs(context.Background(), tc.filter)
//
//			if tc.wantErr {
//				assert.Error(t, err)
//				assert.Nil(t, jobs)
//			} else {
//				assert.NoError(t, err)
//				assert.Len(t, jobs, tc.wantCount)
//				if tc.wantCount > 0 {
//					assert.Equal(t, tc.mockJobs[0].ID, jobs[0].ID)
//				}
//			}
//		})
//	}
//}
//
//func TestGetStatistics(t *testing.T) {
//	testCases := []struct {
//		name        string
//		jobID       string
//		setupMocks  func(*crackingService, *MockRepository)
//		wantErr     bool
//		wantMetrics bool
//	}{
//		{
//			name:  "Get active job metrics",
//			jobID: "active-job",
//			setupMocks: func(s *crackingService, repo *MockRepository) {
//				metrics := &domain.ResourceMetrics{
//					CPUUsage:      50.0,
//					ActiveThreads: 4,
//				}
//				s.metrics.StartCollection("active-job")
//				time.Sleep(100 * time.Millisecond) // Allow metrics to be collected
//			},
//			wantErr:     false,
//			wantMetrics: true,
//		},
//		{
//			name:  "Get completed job metrics",
//			jobID: "completed-job",
//			setupMocks: func(s *crackingService, repo *MockRepository) {
//				repo.On("GetJob", mock.Anything, "completed-job").Return(&domain.CrackingJob{
//					ID: "completed-job",
//					ResourceMetrics: domain.ResourceMetrics{
//						CPUUsage:      0.0,
//						ActiveThreads: 0,
//					},
//				}, nil)
//			},
//			wantErr:     false,
//			wantMetrics: true,
//		},
//		{
//			name:  "Job not found",
//			jobID: "nonexistent",
//			setupMocks: func(s *crackingService, repo *MockRepository) {
//				repo.On("GetJob", mock.Anything, "nonexistent").Return(nil, errors.New("not found"))
//			},
//			wantErr:     true,
//			wantMetrics: false,
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			mockRepo := new(MockRepository)
//			mockHash := new(MockHashService)
//
//			service := NewCrackingService(mockRepo, mockHash, nil)
//			tc.setupMocks(service.(*crackingService), mockRepo)
//
//			metrics, err := service.GetStatistics(context.Background(), tc.jobID)
//
//			if tc.wantErr {
//				assert.Error(t, err)
//				assert.Nil(t, metrics)
//			} else {
//				assert.NoError(t, err)
//				assert.NotNil(t, metrics)
//			}
//		})
//	}
//}
