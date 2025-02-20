package domain

type HashType string
type JobStatus string
type CrackingAlgorithm string
type ProxyType string
type StrengthLevel string

const (
	//Hash types
	HashMD5    HashType = "MD5"
	HashSHA1   HashType = "SHA1"
	HashSHA256 HashType = "SHA256"
	HashSHA512 HashType = "SHA512"
	HashBCRYPT HashType = "BCRYPT"
	HashArgon2 HashType = "ARGON2"
	HashNTLM   HashType = "NTLM"
	HashWPA    HashType = "WPA"

	//Job status
	StatusPending   JobStatus = "PENDING"
	StatusRunning   JobStatus = "RUNNING"
	StatusPaused    JobStatus = "PAUSED"
	StatusComplete  JobStatus = "COMPLETE"
	StatusFailed    JobStatus = "FAILED"
	StatusCancelled JobStatus = "CANCELLED"

	// Cracking Algorithms
	AlgoBruteForce  CrackingAlgorithm = "BRUTE_FORCE"
	AlgoDictionary  CrackingAlgorithm = "DICTIONARY"
	AlgoRainbow     CrackingAlgorithm = "RAINBOW"
	AlgoMask        CrackingAlgorithm = "MASK"
	AlgoHybrid      CrackingAlgorithm = "HYBRID"
	AlgoMarkov      CrackingAlgorithm = "MARKOV"
	AlgoRules       CrackingAlgorithm = "RULES"
	AlgoFingerprint CrackingAlgorithm = "FINGERPRINT"

	// Proxy Types
	ProxyHTTP   ProxyType = "HTTP"
	ProxySOCKS4 ProxyType = "SOCKS4"
	ProxySOCKS5 ProxyType = "SOCKS5"

	// Password Strength Levels
	StrengthVeryWeak   StrengthLevel = "VERY_WEAK"
	StrengthWeak       StrengthLevel = "WEAK"
	StrengthMedium     StrengthLevel = "MEDIUM"
	StrengthStrong     StrengthLevel = "STRONG"
	StrengthVeryStrong StrengthLevel = "VERY_STRONG"
)

var (
	CharsetLower   = "abcdefghijklmnopqrstuvwxyz"
	CharsetUpper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	CharsetDigits  = "0123456789"
	CharsetSpecial = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	CharsetAll     = CharsetLower + CharsetUpper + CharsetDigits + CharsetSpecial
)

type CrackingError string

const (
	ErrInvalidHash     CrackingError = "INVALID_HASH"
	ErrUnsupportedHash CrackingError = "UNSUPPORTED_HASH"
	ErrTimeout         CrackingError = "TIMEOUT"
	ErrResourceLimit   CrackingError = "RESOURCE_LIMIT"
	ErrInvalidWordlist CrackingError = "INVALID_WORDLIST"
)

func (e CrackingError) Error() string {
	return string(e)

}
