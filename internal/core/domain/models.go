package domain

import "time"

type CrackingJob struct {
	ID              string
	TargetHash      string
	HashType        HashType
	Status          JobStatus
	StartTime       time.Time
	EndTime         time.Time
	FoundPassword   string
	Progress        float64
	Algorithm       CrackingAlgorithm
	Settings        CrackingSettings
	ResourceMetrics ResourceMetrics
	AttemptCount    int64
	LastAttempt     time.Time
	ErrorMessage    string
}

type CrackingSettings struct {
	MinLength      int
	MaxLength      int
	CharacterSet   string
	UseWordlist    bool
	WordlistPaths  []string
	CustomRules    []string
	Threads        int
	TimeoutMinutes int
	TargetURL      string
	ProxySettings  *ProxySettings
	Priority       int
	MaxAttempts    int64
	RetryCount     int
}

type ProxySettings struct {
	ProxyURL  string
	Username  string
	Password  string
	ProxyType ProxyType
}

type ResourceMetrics struct {
	CPUUsage       float64
	MemoryUsageMB  int64
	AttemptsPerSec int64
	TotalAttempts  int64
	ActiveThreads  int
	LastUpdated    time.Time
}

type CrackResult struct {
	Password     string
	TimeTaken    time.Duration
	AttemptsUsed int64
	Algorithm    CrackingAlgorithm
	Pattern      string
	Complexity   PasswordComplexity
}

type PasswordComplexity struct {
	Score       int
	Entropy     float64
	TimeToBreak time.Duration
	WeakPoints  []string
	Strength    StrengthLevel
}

type WordlistEntry struct {
	ID        string
	Path      string
	Name      string
	Size      int64
	WordCount int64
	Category  string
	LastUsed  time.Time
}
