package endpoints

// AnnotationRuleConfig annotation rules
type AnnotationRuleConfig struct {
	IgnoreMetrics []string `mapstructure:"ignore_metrics"`
}
