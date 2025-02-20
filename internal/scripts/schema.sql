CREATE TABLE cracking_jobs (
                               id VARCHAR(36) PRIMARY KEY,
                               target_hash VARCHAR(255) NOT NULL,
                               hash_type VARCHAR(50) NOT NULL,
                               status VARCHAR(20) NOT NULL,
                               start_time TIMESTAMP NOT NULL,
                               end_time TIMESTAMP NULL,
                               found_password VARCHAR(255) NULL,
                               progress FLOAT NOT NULL DEFAULT 0,
                               algorithm VARCHAR(50) NOT NULL,
                               settings JSON NOT NULL
);

CREATE TABLE wordlists (
                           id VARCHAR(36) PRIMARY KEY,
                           path VARCHAR(512) NOT NULL,
                           name VARCHAR(255) NOT NULL,
                           size BIGINT NOT NULL,
                           word_count BIGINT NOT NULL,
                           category VARCHAR(50) NOT NULL,
                           last_used TIMESTAMP NOT NULL
);

CREATE TABLE job_metrics (
                             id BIGINT AUTO_INCREMENT PRIMARY KEY,
                             job_id VARCHAR(36) NOT NULL,
                             cpu_usage FLOAT NOT NULL,
                             memory_usage_mb BIGINT NOT NULL,
                             attempts_per_sec BIGINT NOT NULL,
                             total_attempts BIGINT NOT NULL,
                             active_threads INT NOT NULL,
                             timestamp TIMESTAMP NOT NULL,
                             FOREIGN KEY (job_id) REFERENCES cracking_jobs(id)
);
