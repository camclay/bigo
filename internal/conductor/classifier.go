package conductor

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/cammy/bigo/pkg/types"
)

// Classifier determines task complexity and routes to appropriate backends
type Classifier struct {
	patterns map[types.Tier][]Pattern
}

// Pattern represents a classification pattern
type Pattern struct {
	Name    string
	Regex   *regexp.Regexp
	Weight  float64
}

// NewClassifier creates a new task classifier with default patterns
func NewClassifier() *Classifier {
	c := &Classifier{
		patterns: make(map[types.Tier][]Pattern),
	}
	c.initPatterns()
	return c
}

func (c *Classifier) initPatterns() {
	// TRIVIAL patterns - simple edits, formatting, typos
	c.patterns[types.TierTrivial] = []Pattern{
		{Name: "typo", Regex: regexp.MustCompile(`(?i)\b(typo|spelling|spelt|misspell)`), Weight: 0.9},
		{Name: "format", Regex: regexp.MustCompile(`(?i)\b(format|indent|whitespace|spacing)\b`), Weight: 0.8},
		{Name: "comment", Regex: regexp.MustCompile(`(?i)\b(add|update|fix)\s+(a\s+)?comment`), Weight: 0.8},
		{Name: "rename_local", Regex: regexp.MustCompile(`(?i)\brename\s+(the\s+)?(variable|param|local)`), Weight: 0.7},
		{Name: "simple_string", Regex: regexp.MustCompile(`(?i)\b(change|update)\s+(the\s+)?(string|text|message|label)`), Weight: 0.6},
		{Name: "import", Regex: regexp.MustCompile(`(?i)\b(add|remove|fix)\s+(an?\s+)?import`), Weight: 0.7},
	}

	// SIMPLE patterns - straightforward changes
	c.patterns[types.TierSimple] = []Pattern{
		{Name: "add_function", Regex: regexp.MustCompile(`(?i)\badd\s+(a\s+)?(simple\s+)?(function|method|helper)`), Weight: 0.7},
		{Name: "fix_bug_obvious", Regex: regexp.MustCompile(`(?i)\bfix\s+(the\s+)?(bug|issue|error|crash)\s+(in|where|when)`), Weight: 0.6},
		{Name: "update_config", Regex: regexp.MustCompile(`(?i)\b(update|change|modify)\s+(the\s+)?config`), Weight: 0.7},
		{Name: "add_field", Regex: regexp.MustCompile(`(?i)\badd\s+(a\s+)?(new\s+)?(field|property|attribute)`), Weight: 0.6},
		{Name: "simple_validation", Regex: regexp.MustCompile(`(?i)\badd\s+(simple\s+)?validation`), Weight: 0.6},
		{Name: "update_constant", Regex: regexp.MustCompile(`(?i)\b(update|change)\s+(the\s+)?(constant|value|default)`), Weight: 0.7},
	}

	// STANDARD patterns - typical feature work
	c.patterns[types.TierStandard] = []Pattern{
		{Name: "new_feature", Regex: regexp.MustCompile(`(?i)\b(implement|create|build|add)\s+(a\s+)?(new\s+)?feature`), Weight: 0.7},
		{Name: "refactor", Regex: regexp.MustCompile(`(?i)\brefactor\b`), Weight: 0.6},
		{Name: "add_tests", Regex: regexp.MustCompile(`(?i)\b(add|write|create)\s+(unit\s+)?tests?`), Weight: 0.6},
		{Name: "api_endpoint", Regex: regexp.MustCompile(`(?i)\b(add|create|implement)\s+(an?\s+)?(api\s+)?endpoint`), Weight: 0.7},
		{Name: "component", Regex: regexp.MustCompile(`(?i)\b(create|build|add)\s+(a\s+)?(new\s+)?component`), Weight: 0.6},
		{Name: "integration", Regex: regexp.MustCompile(`(?i)\bintegrat(e|ion)\b`), Weight: 0.5},
	}

	// COMPLEX patterns - multi-system changes
	c.patterns[types.TierComplex] = []Pattern{
		{Name: "architecture", Regex: regexp.MustCompile(`(?i)\b(architect|redesign|restructure)`), Weight: 0.8},
		{Name: "migration", Regex: regexp.MustCompile(`(?i)\b(migrat|data\s+migration)`), Weight: 0.8},
		{Name: "cross_cutting", Regex: regexp.MustCompile(`(?i)\b(across|throughout|all)\s+(the\s+)?(codebase|project|system)`), Weight: 0.7},
		{Name: "api_breaking", Regex: regexp.MustCompile(`(?i)\bbreaking\s+change`), Weight: 0.8},
		{Name: "multiple_services", Regex: regexp.MustCompile(`(?i)\bmultiple\s+(service|system|component)s`), Weight: 0.7},
		{Name: "database_schema", Regex: regexp.MustCompile(`(?i)\b(database|db)\s+schema`), Weight: 0.7},
	}

	// CRITICAL patterns - high-risk changes
	c.patterns[types.TierCritical] = []Pattern{
		{Name: "security", Regex: regexp.MustCompile(`(?i)\b(security|vulnerab|exploit|injection|xss|csrf|auth(entication|orization)?)\b`), Weight: 0.9},
		{Name: "payments", Regex: regexp.MustCompile(`(?i)\b(payment|billing|transaction|money|financial)`), Weight: 0.9},
		{Name: "encryption", Regex: regexp.MustCompile(`(?i)\b(encrypt|decrypt|crypto|hash|secret|credential)`), Weight: 0.8},
		{Name: "core_algorithm", Regex: regexp.MustCompile(`(?i)\bcore\s+(algorithm|logic|system)`), Weight: 0.8},
		{Name: "production_data", Regex: regexp.MustCompile(`(?i)\bproduction\s+(data|database|system)`), Weight: 0.9},
		{Name: "user_data", Regex: regexp.MustCompile(`(?i)\b(user|customer|personal)\s+data`), Weight: 0.8},
	}
}

// Classify analyzes a task and returns the classification result
func (c *Classifier) Classify(title, description string) *types.ClassificationResult {
	text := strings.ToLower(title + " " + description)

	result := &types.ClassificationResult{
		Tier:       types.TierStandard, // Default
		Confidence: 0.5,
		Patterns:   []string{},
	}

	// Score each tier
	scores := make(map[types.Tier]float64)
	matchedPatterns := make(map[types.Tier][]string)

	for tier, patterns := range c.patterns {
		for _, p := range patterns {
			if p.Regex.MatchString(text) {
				scores[tier] += p.Weight
				matchedPatterns[tier] = append(matchedPatterns[tier], p.Name)
			}
		}
	}

	// Find highest scoring tier
	maxScore := 0.0
	for tier, score := range scores {
		if score > maxScore {
			maxScore = score
			result.Tier = tier
			result.Patterns = matchedPatterns[tier]
		}
	}

	// Calculate confidence based on score differential
	if maxScore > 0 {
		result.Confidence = min(0.95, 0.5+(maxScore*0.15))
	}

	// Estimate scope from description
	result.EstimatedLines = c.estimateLines(text)
	result.EstimatedFiles = c.estimateFiles(text)

	// Adjust tier based on scope
	result.Tier = c.adjustTierByScope(result.Tier, result.EstimatedLines, result.EstimatedFiles)

	// Set recommended backend
	result.RecommendedBackend = c.recommendBackend(result.Tier)

	// Generate reasoning
	result.Reasoning = c.generateReasoning(result)

	return result
}

func (c *Classifier) estimateLines(text string) int {
	// Heuristics based on task description
	if strings.Contains(text, "single line") || strings.Contains(text, "one line") {
		return 1
	}
	if strings.Contains(text, "few lines") {
		return 5
	}
	if strings.Contains(text, "small") || strings.Contains(text, "minor") {
		return 20
	}
	if strings.Contains(text, "large") || strings.Contains(text, "major") || strings.Contains(text, "significant") {
		return 200
	}
	if strings.Contains(text, "entire") || strings.Contains(text, "complete") || strings.Contains(text, "full") {
		return 500
	}
	return 50 // Default estimate
}

func (c *Classifier) estimateFiles(text string) int {
	if strings.Contains(text, "single file") || strings.Contains(text, "one file") || strings.Contains(text, "this file") {
		return 1
	}
	if strings.Contains(text, "multiple file") || strings.Contains(text, "several file") {
		return 5
	}
	if strings.Contains(text, "across") || strings.Contains(text, "throughout") {
		return 10
	}
	if strings.Contains(text, "codebase") || strings.Contains(text, "project-wide") {
		return 20
	}
	return 2 // Default estimate
}

func (c *Classifier) adjustTierByScope(tier types.Tier, lines, files int) types.Tier {
	// Upgrade tier if scope is large
	if lines > 500 || files > 10 {
		if tier < types.TierComplex {
			return types.TierComplex
		}
	}
	if lines > 200 || files > 5 {
		if tier < types.TierStandard {
			return types.TierStandard
		}
	}

	// Downgrade tier if scope is tiny
	if lines < 10 && files == 1 {
		if tier > types.TierSimple {
			return types.TierSimple
		}
	}

	return tier
}

func (c *Classifier) recommendBackend(tier types.Tier) types.Backend {
	configs := types.DefaultTierConfigs()
	if cfg, ok := configs[tier]; ok {
		return cfg.PrimaryBackend
	}
	return types.BackendClaudeSonnet
}

func (c *Classifier) generateReasoning(result *types.ClassificationResult) string {
	var parts []string

	parts = append(parts, "Tier: "+result.Tier.String())

	if len(result.Patterns) > 0 {
		parts = append(parts, "Matched patterns: "+strings.Join(result.Patterns, ", "))
	}

	parts = append(parts,
		"Estimated scope: ~"+itoa(result.EstimatedLines)+" lines across "+itoa(result.EstimatedFiles)+" file(s)")

	return strings.Join(parts, ". ")
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
