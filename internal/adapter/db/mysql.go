package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"passwordCrakerBackend/internal/core/domain"
	"passwordCrakerBackend/internal/port"
	"time"
)

type mysqlRepository struct {
	db *sql.DB
}

func NewMySQLRepository(dsn string) (port.Repository, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &mysqlRepository{db: db}, nil
}

func (r *mysqlRepository) SaveJob(ctx context.Context, job *domain.CrackingJob) error {
	query := `
        INSERT INTO cracking_jobs (
            id, target_hash, hash_type, status, start_time,
            algorithm, settings, progress
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `

	settings, err := json.Marshal(job.Settings)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query,
		job.ID,
		job.TargetHash,
		job.HashType,
		job.Status,
		job.StartTime,
		job.Algorithm,
		settings,
		job.Progress,
	)
	return err
}

func (r *mysqlRepository) UpdateJob(ctx context.Context, job *domain.CrackingJob) error {
	query := `
        UPDATE cracking_jobs SET
            status = ?,
            end_time = ?,
            found_password = ?,
            progress = ?
        WHERE id = ?
    `

	_, err := r.db.ExecContext(ctx, query,
		job.Status,
		job.EndTime,
		job.FoundPassword,
		job.Progress,
		job.ID,
	)

	return err
}

func (r *mysqlRepository) GetJob(ctx context.Context, jobID string) (*domain.CrackingJob, error) {
	query := `
			SELECT
				id, target_hash, hash_type, status, start_time,
            end_time, found_password, progress, algorithm, settings
        FROM cracking_jobs
        WHERE id = ?
`

	job := &domain.CrackingJob{}
	var settingsJSON []byte
	err := r.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.ID,
		&job.TargetHash,
		&job.HashType,
		&job.Status,
		&job.StartTime,
		&job.EndTime,
		&job.FoundPassword,
		&job.Progress,
		&job.Algorithm,
		&settingsJSON,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(settingsJSON, &job.Settings); err != nil {
		return nil, err
	}

	return job, nil
}

func (r *mysqlRepository) SaveMetrics(ctx context.Context, jobID string, metrics *domain.ResourceMetrics) error {
	query := `
        INSERT INTO job_metrics (
            job_id, cpu_usage, memory_usage_mb, attempts_per_sec,
            total_attempts, active_threads, timestamp
        ) VALUES (?, ?, ?, ?, ?, ?, ?)
    `

	_, err := r.db.ExecContext(ctx, query,
		jobID,
		metrics.CPUUsage,
		metrics.MemoryUsageMB,
		metrics.AttemptsPerSec,
		metrics.TotalAttempts,
		metrics.ActiveThreads,
		metrics.LastUpdated,
	)

	return err
}
func (r *mysqlRepository) DeleteJob(ctx context.Context, jobID string) error {
	query := `DELETE FROM cracking_jobs WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, jobID)
	return err
}

func (r *mysqlRepository) ListJobs(ctx context.Context, filter port.JobFilter) ([]domain.CrackingJob, error) {
	query := `
        SELECT 
            id, target_hash, hash_type, status, start_time,
            end_time, found_password, progress, algorithm, settings
        FROM cracking_jobs
        WHERE 1=1
    `
	args := []interface{}{}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	if filter.HashType != "" {
		query += " AND hash_type = ?"
		args = append(args, filter.HashType)
	}

	if filter.Algorithm != "" {
		query += " AND algorithm = ?"
		args = append(args, filter.Algorithm)
	}

	if filter.StartDate > 0 {
		query += " AND start_time >= FROM_UNIXTIME(?)"
		args = append(args, filter.StartDate)
	}

	if filter.EndDate > 0 {
		query += " AND start_time <= FROM_UNIXTIME(?)"
		args = append(args, filter.EndDate)
	}

	// Add pagination
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)

		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.CrackingJob
	for rows.Next() {
		var job domain.CrackingJob
		var settingsJSON []byte
		err := rows.Scan(
			&job.ID,
			&job.TargetHash,
			&job.HashType,
			&job.Status,
			&job.StartTime,
			&job.EndTime,
			&job.FoundPassword,
			&job.Progress,
			&job.Algorithm,
			&settingsJSON,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(settingsJSON, &job.Settings); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (r *mysqlRepository) SaveWordlist(ctx context.Context, wordlist *domain.WordlistEntry) error {
	query := `
        INSERT INTO wordlists (
            id, path, name, size, word_count, category, last_used
        ) VALUES (?, ?, ?, ?, ?, ?, ?)
    `

	_, err := r.db.ExecContext(ctx, query,
		wordlist.ID,
		wordlist.Path,
		wordlist.Name,
		wordlist.Size,
		wordlist.WordCount,
		wordlist.Category,
		wordlist.LastUsed,
	)
	return err
}

func (r *mysqlRepository) GetWordlists(ctx context.Context) ([]domain.WordlistEntry, error) {
	query := `
        SELECT id, path, name, size, word_count, category, last_used
        FROM wordlists
        ORDER BY last_used DESC
    `

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wordlists []domain.WordlistEntry
	for rows.Next() {
		var wordlist domain.WordlistEntry
		err := rows.Scan(
			&wordlist.ID,
			&wordlist.Path,
			&wordlist.Name,
			&wordlist.Size,
			&wordlist.WordCount,
			&wordlist.Category,
			&wordlist.LastUsed,
		)
		if err != nil {
			return nil, err
		}
		wordlists = append(wordlists, wordlist)
	}

	return wordlists, nil
}
